package integration_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	ormrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/orm/repositories"
	sqlrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/sql/repositories"
	domrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
)

func TestMigrations_ApplySuccessfully(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping integration test in CI")
	}
	tdb := setupPostgres(t)
	defer tdb.Close(t)

	applyMigrations(t, tdb.SQL)

	checks := []string{
		"chats",
		"links",
		"subscriptions",
		"tags",
		"subscription_tags",
	}

	for _, tableName := range checks {
		var exists bool
		err := tdb.SQL.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_name = $1
			)
		`, tableName).Scan(&exists)

		require.NoError(t, err)
		require.True(t, exists, "table %s should exist", tableName)
	}
}

func TestLinkRepositoryScenarios(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping integration test in CI")
	}
	factories := []struct {
		name string
		make func(t *testing.T, tdb *testDB) domrepo.LinkRepository
	}{
		{
			name: "sql",
			make: func(t *testing.T, tdb *testDB) domrepo.LinkRepository {
				log := logger.New(slog.LevelError)
				return sqlrepo.NewSQLChatLinkRepository(tdb.SQL, log)
			},
		},
		{
			name: "orm",
			make: func(t *testing.T, tdb *testDB) domrepo.LinkRepository {
				return ormrepo.NewORMChatLinkRepository(tdb.GORM)
			},
		},
	}

	for _, factory := range factories {
		t.Run(factory.name, func(t *testing.T) {
			tdb := setupPostgres(t)
			defer tdb.Close(t)

			applyMigrations(t, tdb.SQL)

			repo := factory.make(t, tdb)
			ctx := context.Background()

			t.Run("add link", func(t *testing.T) {
				resetDatabase(t, tdb.SQL)
				insertChat(t, tdb.SQL, 123)

				err := repo.Add(ctx, 123, domain.Link{
					URL:           "https://github.com/test/repo",
					Tags:          []string{"go", "backend"},
					LastUpdatedAt: time.Now(),
				})
				require.NoError(t, err)

				links, err := repo.List(ctx, 123)
				require.NoError(t, err)
				require.Len(t, links, 1)
				require.Equal(t, "https://github.com/test/repo", links[0].URL)
				require.ElementsMatch(t, []string{"go", "backend"}, links[0].Tags)
			})

			t.Run("remove link", func(t *testing.T) {
				resetDatabase(t, tdb.SQL)
				insertChat(t, tdb.SQL, 123)

				err := repo.Add(ctx, 123, domain.Link{
					URL:           "https://github.com/test/repo",
					Tags:          []string{"go"},
					LastUpdatedAt: time.Now(),
				})
				require.NoError(t, err)

				err = repo.Remove(ctx, 123, "https://github.com/test/repo")
				require.NoError(t, err)

				links, err := repo.List(ctx, 123)
				require.NoError(t, err)
				require.Len(t, links, 0)
			})

			t.Run("duplicate link does not create second row", func(t *testing.T) {
				resetDatabase(t, tdb.SQL)
				insertChat(t, tdb.SQL, 123)

				link := domain.Link{
					URL:           "https://github.com/test/repo",
					Tags:          []string{"go"},
					LastUpdatedAt: time.Now(),
				}

				err := repo.Add(ctx, 123, link)
				require.NoError(t, err)

				err = repo.Add(ctx, 123, link)
				require.NoError(t, err)

				links, err := repo.List(ctx, 123)
				require.NoError(t, err)
				require.Len(t, links, 1)
			})

			t.Run("remove by tag", func(t *testing.T) {
				resetDatabase(t, tdb.SQL)
				insertChat(t, tdb.SQL, 123)

				err := repo.Add(ctx, 123, domain.Link{
					URL:           "https://github.com/test/repo1",
					Tags:          []string{"go", "work"},
					LastUpdatedAt: time.Now(),
				})
				require.NoError(t, err)

				err = repo.Add(ctx, 123, domain.Link{
					URL:           "https://github.com/test/repo2",
					Tags:          []string{"study"},
					LastUpdatedAt: time.Now(),
				})
				require.NoError(t, err)

				err = repo.Add(ctx, 123, domain.Link{
					URL:           "https://github.com/test/repo3",
					Tags:          []string{"go"},
					LastUpdatedAt: time.Now(),
				})
				require.NoError(t, err)

				removedCount, err := repo.RemoveByTag(ctx, 123, "go")
				require.NoError(t, err)
				require.Equal(t, int64(2), removedCount)

				links, err := repo.List(ctx, 123)
				require.NoError(t, err)
				require.Len(t, links, 1)
				require.Equal(t, "https://github.com/test/repo2", links[0].URL)
			})

			t.Run("list by tag", func(t *testing.T) {
				resetDatabase(t, tdb.SQL)
				insertChat(t, tdb.SQL, 123)

				err := repo.Add(ctx, 123, domain.Link{
					URL:           "https://github.com/test/repo1",
					Tags:          []string{"go"},
					LastUpdatedAt: time.Now(),
				})
				require.NoError(t, err)

				err = repo.Add(ctx, 123, domain.Link{
					URL:           "https://github.com/test/repo2",
					Tags:          []string{"study"},
					LastUpdatedAt: time.Now(),
				})
				require.NoError(t, err)

				links, err := repo.ListByTag(ctx, 123, "go")
				require.NoError(t, err)
				require.Len(t, links, 1)
				require.Equal(t, "https://github.com/test/repo1", links[0].URL)
			})
			t.Run("remove non-existent link returns error", func(t *testing.T) {
				resetDatabase(t, tdb.SQL)
				insertChat(t, tdb.SQL, 123)

				err := repo.Remove(ctx, 123, "https://github.com/test/missing")
				require.Error(t, err)
			})

			t.Run("list by tag returns empty result", func(t *testing.T) {
				resetDatabase(t, tdb.SQL)
				insertChat(t, tdb.SQL, 123)

				err := repo.Add(ctx, 123, domain.Link{
					URL:           "https://github.com/test/repo1",
					Tags:          []string{"go"},
					LastUpdatedAt: time.Now(),
				})
				require.NoError(t, err)

				links, err := repo.ListByTag(ctx, 123, "missing-tag")
				require.NoError(t, err)
				require.Len(t, links, 0)
			})
		})
	}
}
