package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

//go:embed migrate/*.sql
var migrationFS embed.FS

func NewDB(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	needClose := true
	defer func() {
		if needClose {
			db.Close()
		}
	}()

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
