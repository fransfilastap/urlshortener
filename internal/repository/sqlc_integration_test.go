//go:build !short

package repository

import (
	"context"
	"testing"
	"time"

	sqlc "github.com/fransfilastap/urlshortener/internal/db/sqlc"
	"github.com/fransfilastap/urlshortener/internal/domain"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runTestMigrations(dbURL string) error {
	m, err := migrate.New("file://../../db/migrations", dbURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err.Error() != "no change" {
		return err
	}
	return nil
}

func TestSqlcQueries_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, err := SetupPostgresContainer(ctx)
	require.NoError(t, err, "Failed to setup PostgreSQL container")
	defer pgContainer.Teardown(ctx)

	err = runTestMigrations(pgContainer.URI)
	require.NoError(t, err, "Failed to run migrations")

	repo, err := NewPostgresRepository(pgContainer.URI)
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer repo.Close()

	queries := sqlc.New(repo.Pool())

	t.Run("CreateURL", func(t *testing.T) {
		result, err := queries.CreateURL(ctx, sqlc.CreateURLParams{
			Original:         "https://example.com",
			Short:            "test123",
			Title:            pgtypeText("Example Website"),
			CreatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
			ExpiresAt:        pgtype.Timestamp{Time: time.Now().Add(24 * time.Hour), Valid: true},
			Clicks:           0,
			CreatorReference: pgtypeText("ABC"),
			DeletedAt:        pgtype.Timestamp{Valid: false},
		})
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", result.Original)
		assert.Equal(t, "test123", result.Short)
	})

	t.Run("GetURLByShort", func(t *testing.T) {
		result, err := queries.GetURLByShort(ctx, "test123")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", result.Original)
		assert.Equal(t, int64(0), result.Clicks)
	})

	t.Run("GetURLsByCreator", func(t *testing.T) {
		results, err := queries.GetURLsByCreator(ctx, pgtypeText("ABC"))
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("IncrementClicks", func(t *testing.T) {
		clicks, err := queries.IncrementClicks(ctx, "test123")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), clicks)
	})

	t.Run("StoreClick", func(t *testing.T) {
		click, err := queries.StoreClick(ctx, sqlc.StoreClickParams{
			UrlID:     1,
			UrlShort:  "clicktest",
			Ip:        "127.0.0.1",
			Location:  pgtypeText("Unknown"),
			Browser:   pgtypeText("Chrome"),
			Device:    pgtypeText("Desktop"),
			Timestamp: pgtype.Timestamp{Time: time.Now(), Valid: true},
		})
		assert.NoError(t, err)
		_ = click
	})

	t.Run("HasRecentClick", func(t *testing.T) {
		params := sqlc.HasRecentClickParams{
			UrlShort: "clicktest",
			Ip:       "127.0.0.1",
			Browser:  pgtypeText("Chrome"),
			Device:   pgtypeText("Desktop"),
		}
		exists, err := queries.HasRecentClick(ctx, params)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("SoftDeleteURL", func(t *testing.T) {
		rowsAffected, err := queries.SoftDeleteURL(ctx, "test123")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)

		_, err = queries.GetURLByShort(ctx, "test123")
		assert.Error(t, err)
	})
}

func TestPostgresRepository_WithMigrations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	pgContainer, err := SetupPostgresContainer(ctx)
	require.NoError(t, err, "Failed to setup PostgreSQL container")
	defer pgContainer.Teardown(ctx)

	err = runTestMigrations(pgContainer.URI)
	require.NoError(t, err, "Failed to run migrations")

	repo, err := NewPostgresRepository(pgContainer.URI)
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer repo.Close()

	_, err = repo.Pool().Exec(ctx, "DELETE FROM urls")
	require.NoError(t, err, "Failed to clean up existing data")

	t.Run("Create", func(t *testing.T) {
		url := domain.NewURL("https://example.com", "test123", "Example Website", time.Now().Add(24*time.Hour), "ABC")
		createdURL, err := repo.Create(ctx, url)
		assert.NoError(t, err)
		assert.NotNil(t, createdURL)
		assert.Equal(t, "https://example.com", createdURL.Original)
		assert.Equal(t, "test123", createdURL.Short)
		assert.Equal(t, "Example Website", createdURL.Title)
	})

	t.Run("GetByShort", func(t *testing.T) {
		url, err := repo.GetByShort(ctx, "test123")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", url.Original)
		assert.Equal(t, "test123", url.Short)
		assert.Equal(t, int64(0), url.Clicks)
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "test123")
		assert.NoError(t, err)

		_, err = repo.GetByShort(ctx, "test123")
		assert.Equal(t, domain.ErrURLNotFound, err)
	})

	t.Run("DeleteWithCreator", func(t *testing.T) {
		url := domain.NewURL("https://creator-del.com", "creator-del", "Creator Del Test", time.Now().Add(24*time.Hour), "creator-ref")
		_, err := repo.Create(ctx, url)
		assert.NoError(t, err)

		err = repo.DeleteWithCreator(ctx, "creator-del", "creator-ref")
		assert.NoError(t, err)

		err = repo.DeleteWithCreator(ctx, "creator-del", "wrong-ref")
		assert.Error(t, err)
	})

	t.Run("UpdateURLWithCreator", func(t *testing.T) {
		url := domain.NewURL("https://creator-upd.com", "creator-upd", "Creator Upd Test", time.Now().Add(24*time.Hour), "upd-ref")
		_, err := repo.Create(ctx, url)
		assert.NoError(t, err)

		url.Original = "https://creator-upd-v2.com"
		url.Title = "Updated Title"
		err = repo.UpdateURLWithCreator(ctx, "creator-upd", url, "upd-ref")
		assert.NoError(t, err)

		err = repo.UpdateURLWithCreator(ctx, "creator-upd", url, "wrong-ref")
		assert.Error(t, err)
	})
}