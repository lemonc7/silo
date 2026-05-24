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
  s.season_number,
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

-- name: UpsertMagnetEpisode :execrows
INSERT INTO magnet_episodes (magnet_id, episode_id)
VALUES (?1, ?2)
ON CONFLICT(magnet_id, episode_id) DO NOTHING;

-- name: UpsertProfilePriority :execrows
INSERT INTO profile_priorities (profile, priority)
VALUES (?1, ?2)
ON CONFLICT(profile) DO UPDATE SET
  priority = excluded.priority;
