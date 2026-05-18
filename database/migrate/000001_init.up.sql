CREATE TABLE IF NOT EXISTS media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER NOT NULL UNIQUE,
    type TEXT NOT NULL CHECK(type IN ('movie', 'tv', 'anime')),
    title TEXT NOT NULL,
    air_date DATETIME,
    poster_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'wanted'
      CHECK(status IN ('wanted', 'monitoring', 'completed', 'ignored'))
);

CREATE TABLE IF NOT EXISTS seasons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    episode_count INTEGER NOT NULL,
    air_date DATETIME,
    poster_path TEXT NOT NULL,
    UNIQUE(series_id, season_number)
);

CREATE TABLE IF NOT EXISTS episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    season_id INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    episode_number INTEGER NOT NULL,
    air_date DATETIME,
    status TEXT NOT NULL DEFAULT 'pending'
      CHECK(status IN ('pending', 'downloading', 'downloaded')),
    UNIQUE(season_id, episode_number)
);

CREATE TABLE IF NOT EXISTS resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    season_id INTEGER REFERENCES seasons(id) ON DELETE CASCADE,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE,
    magnet_url TEXT NOT NULL UNIQUE,
    seeders INTEGER NOT NULL DEFAULT 0,
    is_chinese INTEGER NOT NULL DEFAULT 0,
    resolution TEXT,
    status TEXT NOT NULL DEFAULT 'available'
      CHECK(status IN ('available', 'downloading', 'downloaded', 'failed'))
);
