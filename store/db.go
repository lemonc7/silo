package store

import (
	"context"
	"database/sql"
	"time"
)

// ── 数据模型 ─────────────────────────────────────

// DownloadRecord 一条下载记录，用于幂等判断：同一集不重复下载。
type DownloadRecord struct {
	ID           int64     `json:"id"`
	TMDBID       int       `json:"tmdb_id"`
	Type         string    `json:"type"` // "movie" | "tv"
	Title        string    `json:"title"`
	Season       int       `json:"season"`
	Episode      int       `json:"episode"`
	MagnetLink   string    `json:"magnet_link"`
	Status       string    `json:"status"` // pending | downloading | completed | failed
	DownloadedAt time.Time `json:"downloaded_at"`
}

// ── Store 接口 ───────────────────────────────────

// Store 本地状态持久化接口。
type Store interface {
	// IsDownloaded 判断某集是否已下载完成。
	IsDownloaded(ctx context.Context, tmdbID int, season, episode int) (bool, error)

	// RecordDownload 记录一次下载。
	RecordDownload(ctx context.Context, r DownloadRecord) error

	// UpdateStatus 更新下载状态。
	UpdateStatus(ctx context.Context, id int64, status string) error

	// GetPending 获取所有未完成的下载（用于重启恢复）。
	GetPending(ctx context.Context) ([]DownloadRecord, error)

	// Close 关闭数据库。
	Close() error
}

// ── SQLiteStore 实现 ────────────────────────────

// SQLiteStore 基于 SQLite 的 Store 实现。
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 打开或创建 SQLite 数据库。
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path) // 需要引入 _ "modernc.org/sqlite" 或 mattn/go-sqlite3
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS downloads (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			tmdb_id     INTEGER NOT NULL,
			type        TEXT    NOT NULL,
			title       TEXT    NOT NULL,
			season      INTEGER DEFAULT 0,
			episode     INTEGER DEFAULT 0,
			magnet_link TEXT    NOT NULL,
			status      TEXT    DEFAULT 'pending',
			downloaded_at TIMESTAMP
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_download_unique
			ON downloads(tmdb_id, season, episode);
	`)
	return err
}

// ── 方法实现 ─────────────────────────────────────

func (s *SQLiteStore) IsDownloaded(ctx context.Context, tmdbID int, season, episode int) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM downloads WHERE tmdb_id=? AND season=? AND episode=? AND status='completed'",
		tmdbID, season, episode,
	).Scan(&count)
	return count > 0, err
}

func (s *SQLiteStore) RecordDownload(ctx context.Context, r DownloadRecord) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO downloads(tmdb_id, type, title, season, episode, magnet_link, status, downloaded_at) VALUES(?,?,?,?,?,?,?,?)",
		r.TMDBID, r.Type, r.Title, r.Season, r.Episode, r.MagnetLink, r.Status, r.DownloadedAt,
	)
	return err
}

func (s *SQLiteStore) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE downloads SET status=? WHERE id=?",
		status, id,
	)
	return err
}

func (s *SQLiteStore) GetPending(ctx context.Context) ([]DownloadRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, tmdb_id, type, title, season, episode, magnet_link, status, downloaded_at FROM downloads WHERE status IN ('pending','downloading')",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		if err := rows.Scan(&r.ID, &r.TMDBID, &r.Type, &r.Title, &r.Season, &r.Episode, &r.MagnetLink, &r.Status, &r.DownloadedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
