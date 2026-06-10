package integration_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/golang-migrate/migrate/v4"
	mpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type testDB struct {
	Container testcontainers.Container
	SQL       *sql.DB
	GORM      *gorm.DB
	DSN       string
}

func setupPostgres(t *testing.T) *testDB {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("12345"),
	)
	require.NoError(t, err)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	var db *sql.DB
	require.Eventually(t, func() bool {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return false
		}
		return db.Ping() == nil
	}, 30*time.Second, 500*time.Millisecond)

	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	return &testDB{
		Container: container,
		SQL:       db,
		GORM:      gdb,
		DSN:       dsn,
	}
}

func (tdb *testDB) Close(t *testing.T) {
	t.Helper()

	if tdb.SQL != nil {
		_ = tdb.SQL.Close()
	}

	if tdb.Container != nil {
		err := tdb.Container.Terminate(context.Background())
		require.NoError(t, err)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)

	dir := filepath.Dir(filename)

	for {
		if _, err := os.Stat(filepath.Join(dir, "migrations")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("project root with migrations directory not found")
		}
		dir = parent
	}
}

func applyMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	driver, err := mpostgres.WithInstance(db, &mpostgres.Config{})
	require.NoError(t, err)

	absMigrationsPath := filepath.Join(projectRoot(t), "migrations", "scrapper")

	wd, err := os.Getwd()
	require.NoError(t, err)

	relMigrationsPath, err := filepath.Rel(wd, absMigrationsPath)
	require.NoError(t, err)

	relMigrationsPath = filepath.ToSlash(relMigrationsPath)

	sourceURL := "file://" + relMigrationsPath

	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres",
		driver,
	)
	require.NoError(t, err)

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}

func insertChat(t *testing.T, db *sql.DB, chatID int64) {
	t.Helper()

	_, err := db.Exec(`INSERT INTO chats (id) VALUES ($1)`, chatID)
	require.NoError(t, err)
}

func resetDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE TABLE subscription_tags, tags, subscriptions, links, chats RESTART IDENTITY CASCADE
	`)
	require.NoError(t, err)
}
