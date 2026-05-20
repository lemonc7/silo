-- -- name: GetOutOfMovies :many
-- SELECT id, title, air_date
-- FROM media
-- WHERE
--   type = 'movie'
--   AND status IN ('wanted', 'monitoring');

-- -- name: GetSeries :many
-- SELECT
--   m.id AS series_id,
--   m.type,
--   m.title,
--   s.id AS season_id,
--   s.season_number,
--   s.air_date
-- FROM media m
-- JOIN seasons s ON m.id = s.series_id
-- WHERE
--   type IN ('tv', 'anime')
--   AND status IN ('wanted', 'monitoring')
-- ORDER BY m.id, s.season_number;


-- name: GetOutOfMedias :many
SELECT id, type, title, air_date
FROM media
WHERE status IN ('wanted', 'monitoring');

-- name: GetOutOfSyncTVs :many
SELECT id, tmdb_id
FROM media
WHERE
  type IN ('tv', 'anime')
  AND status IN ('wanted', 'monitoring');

-- name: GetOutOfSyncSeasons :many
SELECT m.tmdb_id, s.id, s.season_number
FROM seasons s
JOIN media m ON m.id = s.series_id
WHERE
  m.type IN ('tv', 'anime')
  AND m.status IN ('wanted', 'monitoring')
  AND s.episode_count > (
    SELECT COUNT(*) FROM episodes e WHERE e.season_id = s.id
  );

-- name: UpsertMedia :execrows
INSERT INTO media (tmdb_id, type, title, air_date, poster_path)
VALUES (?1, ?2, ?3, ?4, ?5)
ON CONFLICT(tmdb_id) DO UPDATE SET
  title = excluded.title,
  air_date = excluded.air_date,
  poster_path = excluded.poster_path;

-- name: UpsertSeason :execrows
INSERT INTO seasons (series_id, season_number, episode_count, air_date, poster_path)
VALUES (?1, ?2, ?3, ?4, ?5)
ON CONFLICT(series_id, season_number) DO UPDATE SET
    episode_count = excluded.episode_count,
    air_date = excluded.air_date,
    poster_path   = excluded.poster_path;

-- name: UpsertEpisode :execrows
INSERT INTO episodes (season_id, episode_number, air_date)
VALUES (?1, ?2, ?3)
ON CONFLICT(season_id, episode_number) DO UPDATE SET
    air_date = excluded.air_date;

-- name: UpsertSourceLinks :execrows
INSERT INTO sourcelinks (provider, media_id, season_id, detail_path)
VALUES (?1, ?2, ?3, ?4)
ON CONFLICT(provider, detail_path) DO NOTHING;