package repository

import (
	"context"

	"github.com/fransfilastap/urlshortener/internal/domain"
)

type URLRepository interface {
	Create(ctx context.Context, url *domain.URL) (*domain.URL, error)
	GetByShort(ctx context.Context, short string) (*domain.URL, error)
	GetByOriginal(ctx context.Context, original string) (*domain.URL, error)
	GetByCreator(ctx context.Context, creatorReference string) ([]*domain.URL, error)
	IncrementClicks(ctx context.Context, short string) error
	Delete(ctx context.Context, short string) error
	DeleteWithCreator(ctx context.Context, short string, creatorReference string) error
	StoreClick(ctx context.Context, click *domain.Click) error
	GetClicksByShort(ctx context.Context, short string) ([]*domain.Click, error)
	GetClickAnalytics(ctx context.Context, short string) (map[string]interface{}, error)
	HasRecentClick(ctx context.Context, short string, ip string, browser string, device string) (bool, error)
	UpdateURL(ctx context.Context, short string, url *domain.URL) error
	UpdateURLWithCreator(ctx context.Context, short string, url *domain.URL, creatorReference string) error
	LogURLHistory(ctx context.Context, urlID int64, short string, action string, oldValue, newValue interface{}, modifiedBy string) error
}

type CacheRepositoryInterface interface {
	Set(ctx context.Context, url *domain.URL) error
	GetByShort(ctx context.Context, short string) (*domain.URL, error)
	GetByOriginal(ctx context.Context, original string) (*domain.URL, error)
	IncrementClicks(ctx context.Context, short string) error
	Delete(ctx context.Context, short string) error
	Close() error
}