package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lemonc7/silo/config"
	_ "modernc.org/sqlite"
)

//go:embed migrate/*.sql
var migrationFS embed.FS

func NewDB(path string, cfg config.DatabaseConfig) (*sql.DB, error) {
	// 自动创建父目录
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.DSN(path))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	needClose := true
	defer func() {
		if needClose {
			db.Close()
		}
	}()

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	sourceDriver, err := iofs.New(migrationFS, "migrate")
	if err != nil {
		return nil, fmt.Errorf("create iofs driver: %w", err)
	}

	dbDriver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("create db driver: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"sqlite",
		dbDriver,
	)
	if err != nil {
		return nil, fmt.Errorf("create migration instance: %w", err)
	}

	switch err := m.Up(); err {
	case nil, migrate.ErrNoChange:
	case migrate.ErrLocked:
		return nil, fmt.Errorf("database migration locked by another process")
	default:
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	needClose = false

	return db, nil
}
