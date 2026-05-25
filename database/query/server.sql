-- name: GetMedias :many
SELECT * FROM medias;

-- name: GetSeasons :many
SELECT *
FROM seasons
WHERE series_id = ?1;

-- name: GetEpisodes :many
SELECT *
FROM episodes
WHERE season_id = ?1;