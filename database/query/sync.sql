-- name: GetOutOfSyncTVs :many
SELECT id, tmdb_id
FROM medias
WHERE type IN ('tv', 'anime')
  AND status IN ('wanted', 'monitoring');

-- name: GetOutOfSyncSeasons :many
SELECT m.tmdb_id, s.id, s.season_number
FROM seasons s
JOIN medias m ON m.id = s.series_id
WHERE m.type IN ('tv', 'anime')
  AND m.status IN ('wanted', 'monitoring')
  AND s.episode_count > (
    SELECT COUNT(*) FROM episodes e WHERE e.season_id = s.id
  );

-- name: UpsertMedia :execrows
INSERT INTO medias (tmdb_id, type, title, air_date, poster_path)
VALUES (?1, ?2, ?3, ?4, ?5)
ON CONFLICT(tmdb_id, type) DO UPDATE SET
  title = excluded.title,
  air_date = excluded.air_date,
  poster_path = excluded.poster_path;

-- name: UpsertSeason :execrows
INSERT INTO seasons (series_id, season_number, episode_count, air_date, poster_path)
VALUES (?1, ?2, ?3, ?4, ?5)
ON CONFLICT(series_id, season_number) DO UPDATE SET
  episode_count = excluded.episode_count,
  air_date = excluded.air_date,
  poster_path = excluded.poster_path;

-- name: UpsertEpisode :execrows
INSERT INTO episodes (season_id, episode_number, air_date)
VALUES (?1, ?2, ?3)
ON CONFLICT(season_id, episode_number) DO UPDATE SET
  air_date = excluded.air_date;

-- name: GetMoviesWithoutPage :many
SELECT m.id, m.title, m.air_date
FROM medias m
WHERE m.type = 'movie'
  AND NOT EXISTS (
    SELECT 1
    FROM pages p
    WHERE p.provider = ?1
      AND p.media_id = m.id
      AND p.season_id IS NULL
  );

-- name: GetSeasonsWithoutPage :many
SELECT 
  s.id AS season_id,
  s.series_id,
  m.type,
  m.title,
  s.air_date,
  s.season_number
FROM seasons s
JOIN medias m ON m.id = s.series_id
WHERE m.type IN ('tv', 'anime')
  AND NOT EXISTS (
    SELECT 1
    FROM pages p
    WHERE p.provider = ?1
      AND p.media_id = s.series_id
      AND p.season_id = s.id
  );

-- name: UpsertPages :execrows
INSERT INTO pages (provider, media_id, season_id, detail_path)
VALUES (?1, ?2, ?3, ?4)
ON CONFLICT(provider, detail_path) DO NOTHING;

-- name: GetMoviePages :many
SELECT 
  m.id AS media_id,
  p.detail_path
FROM medias m
JOIN pages p ON p.media_id = m.id
WHERE
  m.type = 'movie'
  AND m.status IN ('wanted', 'monitoring')
  AND p.provider = ?1
  AND p.season_id IS NULL;

-- name: GetSeasonPages :many
SELECT
  s.id AS season_id,
  s.series_id AS media_id,
  s.episode_count,
  p.detail_path
FROM seasons s
JOIN medias m ON m.id = s.series_id
JOIN pages p
  ON p.media_id = s.series_id
  AND p.season_id = s.id
WHERE
  m.type IN ('tv', 'anime')
  AND m.status IN ('wanted', 'monitoring')
  AND p.provider = ?1;

-- name: GetBestMagnetOfMovie :one
SELECT
  m.id,
  m.magnet_url
FROM magnets m
LEFT JOIN profile_priorities pp
  ON pp.profile = m.profile
WHERE
  m.media_id = @media_id
  AND m.size_mb >= @min_size_mb
  AND m.size_mb <= @max_size_mb
  AND m.season_id IS NULL
  AND m.status = 'available'
ORDER BY
  pp.priority IS NULL,
  pp.priority ASC,
  m.seeder DESC,
  m.size_mb DESC
LIMIT 1;

-- name: GetPendingMovies :many
SELECT id
FROM medias
WHERE type = 'movie'
  AND status IN ('wanted', 'monitoring');

-- name: GetSeasonDownloadStats :many
SELECT
  s.id AS season_id,
  COUNT(e.id) AS total_count,
  COUNT(CASE WHEN e.status = 'pending' THEN 1 END) AS missing_count
FROM episodes e
JOIN seasons s ON s.id = e.season_id
JOIN medias m ON m.id = s.series_id
WHERE m.type IN ('tv', 'anime')
  AND m.status IN ('wanted', 'monitoring')
GROUP BY s.id
HAVING COUNT(CASE WHEN e.status = 'pending' THEN 1 END) > 0;

-- name: GetSeasonMagnetCandidates :many
SELECT
  m.id,
  m.magnet_url,
  m.seeder,
  COUNT(me.episode_id) AS total_cover_count,
  COUNT(CASE WHEN e.status = 'pending' THEN 1 END) AS missing_hit_count,
  COUNT(CASE WHEN e.status = 'downloaded' THEN 1 END) AS extra_count,
  COALESCE(pp.priority, 9223372036854775807) AS profile_priority
FROM magnets m
JOIN magnet_episodes me ON me.magnet_id = m.id
JOIN episodes e ON e.id = me.episode_id
LEFT JOIN profile_priorities pp
  ON pp.profile = m.profile
WHERE
  m.season_id = @season_id
  AND m.status = 'available'
GROUP BY m.id
HAVING COUNT(CASE WHEN e.status = 'pending' THEN 1 END) > 0;

-- name: CreateDownload :execrows
INSERT INTO downloads (magnet_id)
SELECT @magnet_id
WHERE NOT EXISTS (
  SELECT 1
  FROM downloads
  WHERE magnet_id = @magnet_id
    AND status IN ('queued', 'downloading')
);

-- name: GetQueuedDownloads :many
SELECT
  d.id,
  d.magnet_id,
  m.magnet_url
FROM downloads d
JOIN magnets m ON m.id = d.magnet_id
WHERE d.status = 'queued';

-- name: MarkDownloadStarted :execrows
UPDATE downloads
SET
  qb_hash = ?2,
  status = 'downloading',
  error = NULL
WHERE id = ?1;

-- name: MarkDownloadFailed :execrows
UPDATE downloads
SET
  status = 'failed',
  error = ?2
WHERE id = ?1;

-- name: MarkDownloadCompleted :execrows
UPDATE downloads
SET status = 'completed'
WHERE id = ?1;

-- name: MarkMovieDownloadingByMagnet :execrows
UPDATE medias
SET status = 'downloading'
WHERE id = (
  SELECT media_id
  FROM magnets m
  WHERE m.id = ?1
)
  AND type = 'movie'
  AND status IN ('wanted', 'monitoring');

-- name: MarkEpisodesDownloadingByMagnet :execrows
UPDATE episodes
SET status = 'downloading'
WHERE id IN (
  SELECT episode_id
  FROM magnet_episodes
  WHERE magnet_id = ?1
)
  AND status = 'pending';

-- name: MarkMovieCompletedByMagnet :execrows
UPDATE medias
SET status = 'completed'
WHERE id = (
  SELECT media_id
  FROM magnets m
  WHERE m.id = ?1
)
  AND type = 'movie';

-- name: MarkEpisodesDownloadedByMagnet :execrows
UPDATE episodes
SET status = 'downloaded'
WHERE id IN (
  SELECT episode_id
  FROM magnet_episodes
  WHERE magnet_id = ?1
);


-- name: UpsertMagnets :one
INSERT INTO magnets (
  media_id, 
  season_id, 
  title, 
  magnet_url,
  size_mb,
  seeder,
  profile
) VALUES (
  ?1,
  ?2,
  ?3,
  ?4,
  ?5,
  ?6,
  ?7
)
ON CONFLICT(magnet_url) DO UPDATE SET
  title = excluded.title,
  size_mb = excluded.size_mb,
  seeder = excluded.seeder,
  profile = excluded.profile
RETURNING id;

-- name: UpsertProfilePriority :execrows
INSERT INTO profile_priorities (profile, priority)
VALUES (?1, ?2)
ON CONFLICT(profile) DO UPDATE SET
  priority = excluded.priority;

-- name: UpsertMagnetEpisodeByEpisodeNumber :execrows
INSERT INTO magnet_episodes (magnet_id, episode_id)
SELECT
  ?1,
  e.id
FROM episodes e
WHERE
  e.season_id = ?2
  AND e.episode_number = ?3
ON CONFLICT(magnet_id, episode_id) DO NOTHING;

-- name: UpsertMagnetEpisodeBySeasonID :execrows
INSERT INTO magnet_episodes (magnet_id, episode_id)
SELECT
  ?1,
  e.id
FROM episodes e
WHERE
  e.season_id = ?2
ON CONFLICT(magnet_id, episode_id) DO NOTHING;
