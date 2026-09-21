package psql

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
)

type Storage struct {
	db  *sql.DB
	log *slog.Logger
}

func NewStorage(db *sql.DB, log *slog.Logger) *Storage {
	return &Storage{
		db:  db,
		log: log,
	}
}

func (s *Storage) RunMigrations() error {
	driver, err := postgres.WithInstance(s.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create postgres driver: %w", err)
	}

	src, err := (&file.File{}).Open("file://internal/storage/psql/migrations")
	if err != nil {
		return fmt.Errorf("open migration files: %w", err)
	}

	m, err := migrate.NewWithInstance("file", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	s.log.Info("Applying database migrations...")
	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			s.log.Info("No new migrations to apply")
			return nil
		}
		return fmt.Errorf("m.Up: %w", err)
	}
	s.log.Info("Database migrations applied successfully")

	return nil
}
