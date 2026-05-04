package repository

import (
	"context"
	"testing"
	"time"

	"github.com/fransfilastap/urlshortener/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheRepository_Integration(t *testing.T) {
	ctx := context.Background()
	redisContainer, err := SetupRedisContainer(ctx)
	require.NoError(t, err, "Failed to setup Redis container")
	defer redisContainer.Teardown(ctx)

	ttl := 1 * time.Hour
	repo := NewCacheRepository(redisContainer.URI, "", 0, ttl)
	defer repo.Close()

	url := domain.NewURL("https://example.com", "test123", "Example Website", time.Now().Add(24*time.Hour), "test-user")

	t.Run("Set", func(t *testing.T) {
		err := repo.Set(ctx, url)
		assert.NoError(t, err)
	})

	t.Run("GetByShort", func(t *testing.T) {
		err := repo.Set(ctx, url)
		require.NoError(t, err)

		retrieved, err := repo.GetByShort(ctx, url.Short)
		assert.NoError(t, err)
		assert.Equal(t, url.Original, retrieved.Original)
		assert.Equal(t, url.Short, retrieved.Short)
		assert.Equal(t, url.Title, retrieved.Title)
	})

	t.Run("GetByOriginal", func(t *testing.T) {
		err := repo.Set(ctx, url)
		require.NoError(t, err)

		retrieved, err := repo.GetByOriginal(ctx, url.Original)
		assert.NoError(t, err)
		assert.Equal(t, url.Original, retrieved.Original)
		assert.Equal(t, url.Short, retrieved.Short)
		assert.Equal(t, url.Title, retrieved.Title)
	})

	t.Run("IncrementClicks", func(t *testing.T) {
		url.Clicks = 0
		err := repo.Set(ctx, url)
		require.NoError(t, err)

		err = repo.IncrementClicks(ctx, url.Short)
		assert.NoError(t, err)

		retrieved, err := repo.GetByShort(ctx, url.Short)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), retrieved.Clicks)
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Set(ctx, url)
		require.NoError(t, err)

		err = repo.Delete(ctx, url.Short)
		assert.NoError(t, err)

		_, err = repo.GetByShort(ctx, url.Short)
		assert.Equal(t, domain.ErrURLNotFound, err)
	})
}