package store

import (
	"context"
	"testing"
	"time"

	"github.com/fransfilastap/urlshortener/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheRepository_Integration tests the CacheRepository with a real Redis instance.
// This test uses testcontainers to spin up a Redis container for testing.
func TestCacheRepository_Integration(t *testing.T) {
	// Setup Redis container
	ctx := context.Background()
	redisContainer, err := SetupRedisContainer(ctx)
	require.NoError(t, err, "Failed to setup Redis container")
	defer redisContainer.Teardown(ctx)

	// Create cache repository
	ttl := 1 * time.Hour
	repo := NewCacheRepository(redisContainer.URI, "", 0, ttl)
	defer repo.Close()

	// Test URL for all tests
	url := models.NewURL("https://example.com", "test123", "Example Website", time.Now().Add(24*time.Hour), "test-user")

	// Test setting a URL
	t.Run("Set", func(t *testing.T) {
		err := repo.Set(ctx, url)
		assert.NoError(t, err)
	})

	// Test getting a URL by short code
	t.Run("GetByShort", func(t *testing.T) {
		// First set the URL
		err := repo.Set(ctx, url)
		require.NoError(t, err)

		// Then get it by short code
		retrieved, err := repo.GetByShort(ctx, url.Short)
		assert.NoError(t, err)
		assert.Equal(t, url.Original, retrieved.Original)
		assert.Equal(t, url.Short, retrieved.Short)
		assert.Equal(t, url.Title, retrieved.Title)
	})

	// Test getting a URL by original URL
	t.Run("GetByOriginal", func(t *testing.T) {
		// First set the URL
		err := repo.Set(ctx, url)
		require.NoError(t, err)

		// Then get it by original URL
		retrieved, err := repo.GetByOriginal(ctx, url.Original)
		assert.NoError(t, err)
		assert.Equal(t, url.Original, retrieved.Original)
		assert.Equal(t, url.Short, retrieved.Short)
		assert.Equal(t, url.Title, retrieved.Title)
	})

	// Test incrementing clicks
	t.Run("IncrementClicks", func(t *testing.T) {
		// First set the URL
		url.Clicks = 0
		err := repo.Set(ctx, url)
		require.NoError(t, err)

		// Then increment clicks
		err = repo.IncrementClicks(ctx, url.Short)
		assert.NoError(t, err)

		// Verify clicks were incremented
		retrieved, err := repo.GetByShort(ctx, url.Short)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), retrieved.Clicks)
	})

	// Test deleting a URL
	t.Run("Delete", func(t *testing.T) {
		// First set the URL
		err := repo.Set(ctx, url)
		require.NoError(t, err)

		// Then delete it
		err = repo.Delete(ctx, url.Short)
		assert.NoError(t, err)

		// Verify it's gone
		_, err = repo.GetByShort(ctx, url.Short)
		assert.Equal(t, ErrURLNotFound, err)
	})
}
