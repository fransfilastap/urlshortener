package repository

import (
	"context"
	"testing"
	"time"

	"github.com/fransfilastap/urlshortener/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepository_Integration(t *testing.T) {
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
		url := domain.NewURL("https://example.com/click", "clicktest", "Click Test", time.Now().Add(24*time.Hour), "ABC")
		createdURL, err := repo.Create(ctx, url)
		assert.NoError(t, err)
		assert.NotNil(t, createdURL)

		retrievedURL, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)

		click := domain.NewClick(retrievedURL.ID, "clicktest", "127.0.0.1", "Unknown", "Chrome", "Desktop")
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
		assert.Equal(t, domain.ErrURLNotFound, err)

		err = repo.Delete(ctx, "clicktest")
		assert.NoError(t, err)
	})
}