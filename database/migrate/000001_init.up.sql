CREATE TABLE IF NOT EXISTS medias (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tmdb_id INTEGER NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('movie', 'tv', 'anime')),
    title TEXT NOT NULL,
    air_date DATETIME NOT NULL,
    poster_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'wanted'
      CHECK(status IN ('wanted', 'monitoring', 'downloading', 'completed', 'ignored')),
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
      CHECK(status IN ('pending', 'downloading', 'downloaded', 'ignored')),
    UNIQUE(season_id, episode_number)
);

CREATE TABLE IF NOT EXISTS profile_priorities (
    profile TEXT NOT NULL PRIMARY KEY,
    priority INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL,
    media_id INTEGER NOT NULL REFERENCES medias(id) ON DELETE CASCADE,
    season_id INTEGER REFERENCES seasons(id) ON DELETE CASCADE,
    detail_path TEXT NOT NULL,
    UNIQUE(provider, detail_path)
);

CREATE TABLE IF NOT EXISTS magnets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_id INTEGER NOT NULL REFERENCES medias(id) ON DELETE CASCADE,
    season_id INTEGER REFERENCES seasons(id) ON DELETE CASCADE,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    magnet_url TEXT NOT NULL UNIQUE,
    size_mb REAL NOT NULL DEFAULT 0,
    seeder INTEGER NOT NULL DEFAULT 0,
    profile TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'available'
      CHECK(status IN ('available', 'rejected')),
    CHECK(
      (season_id IS NULL AND episode_id IS NULL)
      OR (season_id IS NOT NULL AND episode_id IS NULL)
      OR (season_id IS NOT NULL AND episode_id IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS downloads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_id INTEGER NOT NULL REFERENCES medias(id) ON DELETE CASCADE,
    season_id INTEGER REFERENCES seasons(id) ON DELETE CASCADE,
    episode_id INTEGER REFERENCES episodes(id) ON DELETE CASCADE,
    magnet_id INTEGER NOT NULL REFERENCES magnets(id) ON DELETE CASCADE,
    qb_hash TEXT UNIQUE,
    status TEXT NOT NULL DEFAULT 'queued'
      CHECK(status IN ('queued', 'downloading', 'completed', 'failed')),
    error TEXT,
    CHECK(
      (season_id IS NULL AND episode_id IS NULL)
      OR (season_id IS NOT NULL AND episode_id IS NULL)
      OR (season_id IS NOT NULL AND episode_id IS NOT NULL)
    )
);
