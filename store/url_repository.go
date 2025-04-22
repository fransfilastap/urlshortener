package store

import (
	"context"
	"errors"

	"github.com/fransfilastap/urlshortener/models"
)

var (
	// ErrURLNotFound is returned when a URL is not found
	ErrURLNotFound = errors.New("url not found")
	// ErrURLExists is returned when a URL with the same short code already exists
	ErrURLExists = errors.New("url with this short code already exists")
	// ErrInvalidURL is returned when the URL is invalid
	ErrInvalidURL = errors.New("invalid url")
	// ErrRecentClick is returned when there's a recent click from the same visitor
	ErrRecentClick = errors.New("recent click from the same visitor")
)

// URLRepository defines the interface for URL storage operations
type URLRepository interface {
	// Create stores a new URL
	Create(ctx context.Context, url *models.URL) error
	// GetByShort retrieves a URL by its short code
	GetByShort(ctx context.Context, short string) (*models.URL, error)
	// GetByOriginal retrieves a URL by its original URL
	GetByOriginal(ctx context.Context, original string) (*models.URL, error)
	// GetByCreator retrieves URLs by their creator reference
	GetByCreator(ctx context.Context, creatorReference string) ([]*models.URL, error)
	// IncrementClicks increments the click count for a URL
	IncrementClicks(ctx context.Context, short string) error
	// Delete removes a URL
	Delete(ctx context.Context, short string) error
	// DeleteWithCreator soft deletes a URL if the creator_reference matches
	DeleteWithCreator(ctx context.Context, short string, creatorReference string) error
	// StoreClick stores click analytics data
	StoreClick(ctx context.Context, click *models.Click) error
	// GetClicksByShort retrieves click analytics data for a URL
	GetClicksByShort(ctx context.Context, short string) ([]*models.Click, error)
	// GetClickAnalytics retrieves aggregated click analytics data for a URL
	GetClickAnalytics(ctx context.Context, short string) (map[string]interface{}, error)
	// HasRecentClick checks if there's a recent click from the same visitor
	HasRecentClick(ctx context.Context, short string, ip string, browser string, device string) (bool, error)
	// UpdateURL updates an existing URL
	UpdateURL(ctx context.Context, short string, url *models.URL) error
	// UpdateURLWithCreator updates an existing URL if the creator_reference matches
	UpdateURLWithCreator(ctx context.Context, short string, url *models.URL, creatorReference string) error
	// LogURLHistory logs a URL modification
	LogURLHistory(ctx context.Context, urlID int64, short string, action string, oldValue, newValue interface{}, modifiedBy string) error
}
