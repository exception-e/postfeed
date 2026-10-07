package psql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	if _, err := testcontainers.NewDockerProvider(); err != nil {
		println("integration: Docker unavailable, skipping:", err.Error())
		os.Exit(0) // exit 0, чтобы `go test ./...` оставался зелёным
	}

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic(err)
	}
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	testDB, err = sql.Open("pgx", connStr)
	if err != nil {
		panic(err)
	}
	defer testDB.Close()

	if err := testDB.PingContext(ctx); err != nil {
		panic(err)
	}

	if err := runMigrations(testDB); err != nil {
		panic(err)
	}

	code := m.Run()
	pgContainer.Terminate(ctx)
	testDB.Close()

	os.Exit(code)
}

func runMigrations(db *sql.DB) error { //TODO с миграциями
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create postgres driver: %w", err)
	}

	src, err := (&file.File{}).Open("file://migrations")
	if err != nil {
		return fmt.Errorf("open migration files: %w", err)
	}

	m, err := migrate.NewWithInstance("file", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err = m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("m.Up: %w", err)
	}

	return nil
}

func cleanupDB(t *testing.T) {
	t.Helper()
	_, err := testDB.ExecContext(context.Background(),
		"TRUNCATE posts, comments RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}
