CREATE TABLE IF NOT EXISTS media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER NOT NULL UNIQUE,
    type TEXT NOT NULL CHECK(type IN ('movie', 'tv', 'anime')),
    title TEXT NOT NULL,
    year INTEGER NOT NULL DEFAULT 0,
    poster_path TEXT NOT NULL DEFAULT ''
);
