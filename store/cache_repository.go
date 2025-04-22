package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/fransfilastap/urlshortener/models"
	"github.com/redis/go-redis/v9"
)

// CacheRepositoryInterface defines the interface for cache operations
type CacheRepositoryInterface interface {
	// Set stores a URL in the cache
	Set(ctx context.Context, url *models.URL) error
	// GetByShort retrieves a URL by its short code from cache
	GetByShort(ctx context.Context, short string) (*models.URL, error)
	// GetByOriginal retrieves a URL by its original URL from cache
	GetByOriginal(ctx context.Context, original string) (*models.URL, error)
	// IncrementClicks increments the click count for a URL in cache
	IncrementClicks(ctx context.Context, short string) error
	// Delete removes a URL from cache
	Delete(ctx context.Context, short string) error
	// Close closes the cache connection
	Close() error
}

// CacheRepository implements caching for URLs using Valkey/Redis
type CacheRepository struct {
	client *redis.Client
	ttl    time.Duration
}

// Ensure CacheRepository implements CacheRepositoryInterface
var _ CacheRepositoryInterface = (*CacheRepository)(nil)

// NewCacheRepository creates a new cache repository
func NewCacheRepository(addr, password string, db int, ttl time.Duration) *CacheRepository {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &CacheRepository{
		client: client,
		ttl:    ttl,
	}
}

// Set stores a URL in the cache
func (c *CacheRepository) Set(ctx context.Context, url *models.URL) error {
	data, err := json.Marshal(url)
	if err != nil {
		return err
	}

	// Cache by short URL
	err = c.client.Set(ctx, "short:"+url.Short, data, c.ttl).Err()
	if err != nil {
		return err
	}

	// Also cache by original URL
	return c.client.Set(ctx, "original:"+url.Original, data, c.ttl).Err()
}

// GetByShort retrieves a URL by its short code from cache
func (c *CacheRepository) GetByShort(ctx context.Context, short string) (*models.URL, error) {
	data, err := c.client.Get(ctx, "short:"+short).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}

	var url models.URL
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}

	// Check if URL has expired
	if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
		c.client.Del(ctx, "short:"+short, "original:"+url.Original)
		return nil, ErrURLNotFound
	}

	return &url, nil
}

// GetByOriginal retrieves a URL by its original URL from cache
func (c *CacheRepository) GetByOriginal(ctx context.Context, original string) (*models.URL, error) {
	data, err := c.client.Get(ctx, "original:"+original).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}

	var url models.URL
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}

	// Check if URL has expired
	if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
		c.client.Del(ctx, "short:"+url.Short, "original:"+original)
		return nil, ErrURLNotFound
	}

	return &url, nil
}

// IncrementClicks increments the click count for a URL in cache
func (c *CacheRepository) IncrementClicks(ctx context.Context, short string) error {
	// Get the URL from cache
	url, err := c.GetByShort(ctx, short)
	if err != nil {
		return err
	}

	// Increment clicks
	url.Clicks++

	// Update cache
	return c.Set(ctx, url)
}

// Delete removes a URL from cache
func (c *CacheRepository) Delete(ctx context.Context, short string) error {
	// Get the URL to also delete the original key
	url, err := c.GetByShort(ctx, short)
	if err != nil && !errors.Is(err, ErrURLNotFound) {
		return err
	}

	// Delete short URL key
	if err := c.client.Del(ctx, "short:"+short).Err(); err != nil {
		return err
	}

	// If URL was found, also delete original URL key
	if url != nil {
		return c.client.Del(ctx, "original:"+url.Original).Err()
	}

	return nil
}

// Close closes the cache connection
func (c *CacheRepository) Close() error {
	return c.client.Close()
}
