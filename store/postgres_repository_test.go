package store

import (
	"context"
	"testing"
	"time"

	"github.com/fransfilastap/urlshortener/models"
	"github.com/stretchr/testify/assert"
)

// TestPostgresRepository_Integration tests the PostgresRepository with a real database.
// Note: This is an integration test and requires a PostgreSQL database to be running.
// It will be skipped if the database connection fails.
func TestPostgresRepository_Integration(t *testing.T) {
	// Skip this test in CI environments or when no database is available
	t.Skip("Skipping integration test")

	// Try to connect to the database
	repo, err := NewPostgresRepository("postgres://postgres:postgres@localhost:5432/urlshortener_test?sslmode=disable")
	if err != nil {
		t.Skip("Skipping integration test, database not available:", err)
	}
	defer repo.Close()

	// Initialize schema
	ctx := context.Background()
	err = repo.InitSchema(ctx)
	assert.NoError(t, err)

	// Clean up any existing data
	_, err = repo.pool.Exec(ctx, "DELETE FROM urls")
	assert.NoError(t, err)

	// Test creating a URL
	t.Run("Create", func(t *testing.T) {
		url := models.NewURL("https://example.com", "test123", "Example Website", time.Now().Add(24*time.Hour), "ABC")
		err := repo.Create(ctx, url)
		assert.NoError(t, err)
	})

	// Test getting a URL by short code
	t.Run("GetByShort", func(t *testing.T) {
		url, err := repo.GetByShort(ctx, "test123")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", url.Original)
		assert.Equal(t, "test123", url.Short)
		assert.Equal(t, int64(0), url.Clicks)
	})

	// Test getting a URL by original URL
	t.Run("GetByOriginal", func(t *testing.T) {
		url, err := repo.GetByOriginal(ctx, "https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, "test123", url.Short)
	})

	// Test incrementing clicks
	t.Run("IncrementClicks", func(t *testing.T) {
		err := repo.IncrementClicks(ctx, "test123")
		assert.NoError(t, err)

		url, err := repo.GetByShort(ctx, "test123")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), url.Clicks)
	})

	// Test storing click analytics
	t.Run("StoreClick", func(t *testing.T) {
		// Create a new URL for testing
		url := models.NewURL("https://example.com/click", "clicktest", "Click Test", time.Now().Add(24*time.Hour), "ABC")
		err := repo.Create(ctx, url)
		assert.NoError(t, err)

		// Get the URL to get its ID
		retrievedURL, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)

		// Create a click
		click := models.NewClick(retrievedURL.ID, "clicktest", "127.0.0.1", "Unknown", "Chrome", "Desktop")
		err = repo.StoreClick(ctx, click)
		assert.NoError(t, err)
	})

	// Test getting clicks by short code
	t.Run("GetClicksByShort", func(t *testing.T) {
		clicks, err := repo.GetClicksByShort(ctx, "clicktest")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(clicks), 1)
		assert.Equal(t, "clicktest", clicks[0].URLShort)
	})

	// Test checking for recent clicks
	t.Run("HasRecentClick", func(t *testing.T) {
		hasRecent, err := repo.HasRecentClick(ctx, "clicktest", "127.0.0.1", "Chrome", "Desktop")
		assert.NoError(t, err)
		assert.True(t, hasRecent)
	})

	// Test updating a URL
	t.Run("UpdateURL", func(t *testing.T) {
		// Get the URL to update
		url, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)

		// Update the URL
		url.Title = "Updated Title"
		url.Original = "https://example.com/updated"
		err = repo.UpdateURL(ctx, "clicktest", url)
		assert.NoError(t, err)

		// Verify the update
		updatedURL, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)
		assert.Equal(t, "Updated Title", updatedURL.Title)
		assert.Equal(t, "https://example.com/updated", updatedURL.Original)
	})

	// Test logging URL history
	t.Run("LogURLHistory", func(t *testing.T) {
		// Get the URL
		url, err := repo.GetByShort(ctx, "clicktest")
		assert.NoError(t, err)

		// Log a history entry
		oldValue := map[string]string{"title": "Old Title"}
		newValue := map[string]string{"title": "New Title"}
		err = repo.LogURLHistory(ctx, url.ID, "clicktest", "update", oldValue, newValue, "test-user")
		assert.NoError(t, err)
	})

	// Test getting click analytics
	t.Run("GetClickAnalytics", func(t *testing.T) {
		analytics, err := repo.GetClickAnalytics(ctx, "clicktest")
		assert.NoError(t, err)
		assert.NotNil(t, analytics)
		assert.Contains(t, analytics, "total_clicks")
	})

	// Test deleting a URL
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "test123")
		assert.NoError(t, err)

		_, err = repo.GetByShort(ctx, "test123")
		assert.Equal(t, ErrURLNotFound, err)

		// Also delete the click test URL
		err = repo.Delete(ctx, "clicktest")
		assert.NoError(t, err)
	})
}

// TestPostgresRepository_Unit tests the PostgresRepository with a mock database.
func TestPostgresRepository_Unit(t *testing.T) {
	// This is a placeholder for unit tests that would use a mock database.
	// In a real implementation, you would use a library like sqlmock to mock the database.
	t.Skip("Unit tests for PostgresRepository not implemented")
}
