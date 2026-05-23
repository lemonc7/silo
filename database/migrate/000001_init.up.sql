CREATE TABLE IF NOT EXISTS medias (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('movie', 'tv', 'anime')),
    title TEXT NOT NULL,
    air_date DATETIME NOT NULL,
    poster_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'wanted'
      CHECK(status IN ('wanted', 'monitoring', 'completed', 'ignored')),
    UNIQUE(tmdb_id, type)
);

CREATE TABLE IF NOT EXISTS seasons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL REFERENCES medias(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    episode_count INTEGER NOT NULL,
    air_date DATETIME NOT NULL,
    poster_path TEXT NOT NULL,
    UNIQUE(series_id, season_number)
);

CREATE TABLE IF NOT EXISTS episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    season_id INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    episode_number INTEGER NOT NULL,
    air_date DATETIME NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
      CHECK(status IN ('pending', 'downloading', 'downloaded')),
    UNIQUE(season_id, episode_number)
);

CREATE TABLE IF NOT EXISTS pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    media_id INTEGER NOT NULL REFERENCES medias(id) ON DELETE CASCADE,
    season_id INTEGER REFERENCES seasons(id) ON DELETE CASCADE,
    detail_path TEXT NOT NULL,
    UNIQUE(provider, detail_path)
);