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
  m.id,
  p.detail_path
FROM pages p
JOIN medias m ON m.id = p.media_id
WHERE
  m.type = 'movie'
  AND m.status IN ('wanted', 'monitoring')
  AND p.provider = ?1
  AND p.season_id IS NULL;
