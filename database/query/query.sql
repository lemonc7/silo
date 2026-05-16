-- name: UpsertMedia :execrows
INSERT INTO media (tmdb_id, type, title, air_date, poster_path)
VALUES (?1, ?2, ?3, ?4, ?5)
ON CONFLICT(tmdb_id) DO UPDATE SET
    type       = excluded.type,
    title      = excluded.title,
    air_date   = excluded.air_date,
    poster_path = excluded.poster_path;

-- name: GetMediaID :one
SELECT id FROM media WHERE tmdb_id = ?1;

-- name: GetTVMedia :many
SELECT id, tmdb_id FROM media WHERE type IN ('tv', 'anime');

-- name: UpsertSeason :execrows
INSERT INTO seasons (series_id, season_number, episode_count, air_date, poster_path)
VALUES (?1, ?2, ?3, ?4, ?5)
ON CONFLICT(series_id, season_number) DO UPDATE SET
    episode_count = excluded.episode_count,
    air_date      = excluded.air_date,
    poster_path   = excluded.poster_path;

-- name: GetSeasonID :one
SELECT id FROM seasons WHERE series_id = ?1 AND season_number = ?2;

-- name: CountEpisodes :one
SELECT COUNT(*) FROM episodes
WHERE season_id IN (SELECT id FROM seasons WHERE series_id = ?1 AND season_number = ?2);

-- name: UpsertEpisode :execrows
INSERT INTO episodes (season_id, episode_number, air_date, status)
VALUES (?1, ?2, ?3, 'pending')
ON CONFLICT(season_id, episode_number) DO UPDATE SET
    air_date = excluded.air_date;

-- name: GetPendingEpisodes :many
SELECT e.id, e.season_id, e.episode_number, e.air_date, e.status,
       s.season_number, s.series_id
FROM episodes e
JOIN seasons s ON e.season_id = s.id
WHERE e.air_date <= date('now')
  AND e.status = 'pending'
ORDER BY e.air_date ASC;

-- name: MarkEpisodeDownloaded :exec
UPDATE episodes SET status = 'downloaded' WHERE id = ?1;
