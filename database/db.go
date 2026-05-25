package database

import (
	"database/sql"
	"embed"
	"errors"
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
		return nil, fmt.Errorf("创建db父目录: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.DSN(path))
	if err != nil {
		return nil, fmt.Errorf("打开数据库文件: %w", err)
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
		return nil, fmt.Errorf("ping数据库: %w", err)
	}

	sourceDriver, err := iofs.New(migrationFS, "migrate")
	if err != nil {
		return nil, fmt.Errorf("创建iofs驱动: %w", err)
	}

	dbDriver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("创建db驱动: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"sqlite",
		dbDriver,
	)
	if err != nil {
		return nil, fmt.Errorf("创建schema迁移实例: %w", err)
	}

	if err := m.Up(); err != nil {
		switch {
		case errors.Is(err, migrate.ErrNoChange):
		case errors.Is(err, migrate.ErrLocked):
			return nil, fmt.Errorf("数据库迁移被其他进程锁定: %w", err)
		default:
			return nil, fmt.Errorf("迁移数据库: %w", err)
		}
	}

	needClose = false

	return db, nil
}
