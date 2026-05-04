//go:build !short

package store

import (
	"context"
	"testing"
	"time"

	sqlc "github.com/fransfilastap/urlshortener/internal/db/sqlc"
	"github.com/fransfilastap/urlshortener/models"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runTestMigrations(dbURL string) error {
	m, err := migrate.New("file://../db/migrations", dbURL)
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

	t.Run("GetURLByOriginal", func(t *testing.T) {
		result, err := queries.GetURLByOriginal(ctx, "https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, "test123", result.Short)
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

	t.Run("UpdateURL", func(t *testing.T) {
		result, err := queries.UpdateURL(ctx, sqlc.UpdateURLParams{
			Original:  "https://example.com/updated",
			Title:    pgtypeText("Updated Title"),
			ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(48 * time.Hour), Valid: true},
			Short:    "test123",
		})
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/updated", result.Original)
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

	t.Run("GetClicksByShort", func(t *testing.T) {
		clicks, err := queries.GetClicksByShort(ctx, "clicktest")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(clicks), 1)
	})

	t.Run("GetClickAnalytics", func(t *testing.T) {
		totalClicks, err := queries.GetTotalClicks(ctx, "clicktest")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, totalClicks, int64(1))

		devices, err := queries.GetClicksByDevice(ctx, "clicktest")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(devices), 1)

		browsers, err := queries.GetClicksByBrowser(ctx, "clicktest")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(browsers), 1)

		locations, err := queries.GetClicksByLocation(ctx, "clicktest")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(locations), 1)
	})

	t.Run("LogURLHistory", func(t *testing.T) {
		oldValue := []byte(`{"title": "Old Title"}`)
		newValue := []byte(`{"title": "New Title"}`)

		history, err := queries.LogURLHistory(ctx, sqlc.LogURLHistoryParams{
			UrlID:      1,
			UrlShort:   "test123",
			Action:     "update",
			OldValue:   oldValue,
			NewValue:   newValue,
			ModifiedBy: pgtypeText("test-user"),
		})
		assert.NoError(t, err)
		assert.Equal(t, "update", history.Action)
	})

	t.Run("SoftDeleteURL", func(t *testing.T) {
		rowsAffected, err := queries.SoftDeleteURL(ctx, "test123")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)

		_, err = queries.GetURLByShort(ctx, "test123")
		assert.Error(t, err)
	})

	t.Run("SoftDeleteURLWithCreator", func(t *testing.T) {
		_, err := queries.CreateURL(ctx, sqlc.CreateURLParams{
			Original:         "https://creator-test.com",
			Short:           "creator-test",
			Title:            pgtypeText("Creator Test"),
			CreatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
			ExpiresAt:        pgtype.Timestamp{Valid: false},
			Clicks:           0,
			CreatorReference: pgtypeText("creator1"),
			DeletedAt:        pgtype.Timestamp{Valid: false},
		})
		assert.NoError(t, err)

		rowsAffected, err := queries.SoftDeleteURLWithCreator(ctx, sqlc.SoftDeleteURLWithCreatorParams{
			Short:            "creator-test",
			CreatorReference: pgtypeText("creator1"),
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)
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
		url := models.NewURL("https://example.com", "test123", "Example Website", time.Now().Add(24*time.Hour), "ABC")
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

	t.Run("GetByOriginal", func(t *testing.T) {
		url, err := repo.GetByOriginal(ctx, "https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, "test123", url.Short)
	})

	t.Run("IncrementClicks", func(t *testing.T) {
		err := repo.IncrementClicks(ctx, "test123")
		assert.NoError(t, err)

		url, err := repo.GetByShort(ctx, "test123")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), url.Clicks)
	})

	t.Run("StoreClick", func(t *testing.T) {
		clickTestURL := models.NewURL("https://example.com/click", "clicktest", "Click Test", time.Now().Add(24*time.Hour), "ABC")
		createdURL, err := repo.Create(ctx, clickTestURL)
		assert.NoError(t, err)
		assert.NotNil(t, createdURL)

		retrievedURL, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)

		click := models.NewClick(retrievedURL.ID, "clicktest", "127.0.0.1", "Unknown", "Chrome", "Desktop")
		err = repo.StoreClick(ctx, click)
		assert.NoError(t, err)
	})

	t.Run("GetClicksByShort", func(t *testing.T) {
		clicks, err := repo.GetClicksByShort(ctx, "clicktest")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(clicks), 1)
		assert.Equal(t, "clicktest", clicks[0].URLShort)
	})

	t.Run("HasRecentClick", func(t *testing.T) {
		hasRecent, err := repo.HasRecentClick(ctx, "clicktest", "127.0.0.1", "Chrome", "Desktop")
		assert.NoError(t, err)
		assert.True(t, hasRecent)
	})

	t.Run("UpdateURL", func(t *testing.T) {
		url, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)

		url.Title = "Updated Title"
		url.Original = "https://example.com/updated"
		err = repo.UpdateURL(ctx, "clicktest", url)
		assert.NoError(t, err)

		updatedURL, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)
		assert.Equal(t, "Updated Title", updatedURL.Title)
		assert.Equal(t, "https://example.com/updated", updatedURL.Original)
	})

	t.Run("LogURLHistory", func(t *testing.T) {
		url, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)

		oldValue := map[string]string{"title": "Old Title"}
		newValue := map[string]string{"title": "New Title"}
		err = repo.LogURLHistory(ctx, url.ID, "clicktest", "update", oldValue, newValue, "test-user")
		assert.NoError(t, err)
	})

	t.Run("GetClickAnalytics", func(t *testing.T) {
		analytics, err := repo.GetClickAnalytics(ctx, "clicktest")
		assert.NoError(t, err)
		assert.NotNil(t, analytics)
		assert.Contains(t, analytics, "total_clicks")
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "test123")
		assert.NoError(t, err)

		_, err = repo.GetByShort(ctx, "test123")
		assert.Equal(t, ErrURLNotFound, err)

		err = repo.Delete(ctx, "clicktest")
		assert.NoError(t, err)
	})

	t.Run("DeleteWithCreator", func(t *testing.T) {
		url := models.NewURL("https://creator-del.com", "creator-del", "Creator Del Test", time.Now().Add(24*time.Hour), "creator-ref")
		_, err := repo.Create(ctx, url)
		assert.NoError(t, err)

		err = repo.DeleteWithCreator(ctx, "creator-del", "creator-ref")
		assert.NoError(t, err)

		err = repo.DeleteWithCreator(ctx, "creator-del", "wrong-ref")
		assert.Error(t, err)
	})

	t.Run("UpdateURLWithCreator", func(t *testing.T) {
		url := models.NewURL("https://creator-upd.com", "creator-upd", "Creator Upd Test", time.Now().Add(24*time.Hour), "upd-ref")
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