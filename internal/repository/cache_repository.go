package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/fransfilastap/urlshortener/internal/domain"
	"github.com/redis/go-redis/v9"
)

type CacheRepository struct {
	client *redis.Client
	ttl    time.Duration
}

var _ CacheRepositoryInterface = (*CacheRepository)(nil)

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

func (c *CacheRepository) Set(ctx context.Context, url *domain.URL) error {
	data, err := json.Marshal(url)
	if err != nil {
		return err
	}

	err = c.client.Set(ctx, "short:"+url.Short, data, c.ttl).Err()
	if err != nil {
		return err
	}

	return c.client.Set(ctx, "original:"+url.Original, data, c.ttl).Err()
}

func (c *CacheRepository) GetByShort(ctx context.Context, short string) (*domain.URL, error) {
	data, err := c.client.Get(ctx, "short:"+short).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrURLNotFound
		}
		return nil, err
	}

	var url domain.URL
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}

	if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
		c.client.Del(ctx, "short:"+short, "original:"+url.Original)
		return nil, domain.ErrURLNotFound
	}

	return &url, nil
}

func (c *CacheRepository) GetByOriginal(ctx context.Context, original string) (*domain.URL, error) {
	data, err := c.client.Get(ctx, "original:"+original).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrURLNotFound
		}
		return nil, err
	}

	var url domain.URL
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}

	if !url.ExpiresAt.IsZero() && url.ExpiresAt.Before(time.Now()) {
		c.client.Del(ctx, "short:"+url.Short, "original:"+original)
		return nil, domain.ErrURLNotFound
	}

	return &url, nil
}

func (c *CacheRepository) IncrementClicks(ctx context.Context, short string) error {
	url, err := c.GetByShort(ctx, short)
	if err != nil {
		return err
	}

	url.Clicks++

	return c.Set(ctx, url)
}

func (c *CacheRepository) Delete(ctx context.Context, short string) error {
	url, err := c.GetByShort(ctx, short)
	if err != nil && !errors.Is(err, domain.ErrURLNotFound) {
		return err
	}

	if err := c.client.Del(ctx, "short:"+short).Err(); err != nil {
		return err
	}

	if url != nil {
		return c.client.Del(ctx, "original:"+url.Original).Err()
	}

	return nil
}

func (c *CacheRepository) Close() error {
	return c.client.Close()
}