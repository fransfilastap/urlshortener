package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/fransfilastap/urlshortener/models"
	"github.com/rs/zerolog/log"
)

// URLService provides URL shortening and retrieval services
type URLService struct {
	db    URLRepository
	cache CacheRepositoryInterface
}

// NewURLService creates a new URL service
func NewURLService(db URLRepository, cache CacheRepositoryInterface) *URLService {
	return &URLService{
		db:    db,
		cache: cache,
	}
}

// CreateShortURL creates a new short URL
func (s *URLService) CreateShortURL(ctx context.Context, originalURL string, customShort string, title string, expireAfter time.Duration, creatorReference string) (*models.URL, error) {
	log.Debug().
		Str("original_url", originalURL).
		Str("custom_short", customShort).
		Str("title", title).
		Dur("expire_after", expireAfter).
		Str("creator_reference", creatorReference).
		Msg("Creating short URL")

	// Validate URL
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		log.Error().Err(err).Str("url", originalURL).Msg("Invalid URL format")
		return nil, ErrInvalidURL
	}

	// Generate short URL if not provided
	short := customShort
	if short == "" {
		var err error
		log.Debug().Msg("No custom short code provided, generating random code")
		short, err = s.generateShortURL(6)
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate short URL")
			return nil, err
		}
	} else {
		// Check if custom short URL already exists
		_, err := s.GetByShort(ctx, short)
		if err == nil {
			log.Error().Str("custom_short", short).Msg("Custom short code already in use")
			return nil, ErrURLExists
		} else if !errors.Is(err, ErrURLNotFound) {
			log.Error().Err(err).Str("custom_short", short).Msg("Error checking if custom short code exists")
			return nil, err
		}
	}

	// Set expiration time if provided
	var expiresAt time.Time
	if expireAfter > 0 {
		expiresAt = time.Now().Add(expireAfter)
		log.Debug().Time("expires_at", expiresAt).Msg("Setting URL expiration time")
	}

	// Create URL
	newURL := models.NewURL(originalURL, short, title, expiresAt, creatorReference)

	// Save to database
	log.Debug().Str("short", short).Msg("Saving URL to database")
	createdURL, err := s.db.Create(ctx, newURL)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to save URL to database")
		return nil, err
	}

	// Cache the URL
	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Caching URL")
		err := s.cache.Set(ctx, createdURL)
		if err != nil {
			return nil, err
		}
	}

	log.Info().
		Str("original_url", originalURL).
		Str("short", short).
		Time("expires_at", expiresAt).
		Int64("id", createdURL.ID).
		Msg("Short URL created successfully")

	return createdURL, nil
}

// GetByShort retrieves a URL by its short code
func (s *URLService) GetByShort(ctx context.Context, short string) (*models.URL, error) {
	log.Debug().Str("short", short).Msg("Getting URL by short code")

	// Try to get from cache first
	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Checking cache for URL")
		foundURL, err := s.cache.GetByShort(ctx, short)
		if err == nil {
			log.Debug().Str("short", short).Msg("URL found in cache")
			return foundURL, nil
		} else if !errors.Is(err, ErrURLNotFound) {
			log.Error().Err(err).Str("short", short).Msg("Cache error when getting URL by short code")
		} else {
			log.Debug().Str("short", short).Msg("URL not found in cache, checking database")
		}
	}

	// Get from database
	log.Debug().Str("short", short).Msg("Getting URL from database")
	urlRecord, err := s.db.GetByShort(ctx, short)
	if err != nil {
		if errors.Is(err, ErrURLNotFound) {
			log.Debug().Str("short", short).Msg("URL not found in database")
		} else {
			log.Error().Err(err).Str("short", short).Msg("Database error when getting URL by short code")
		}
		return nil, err
	}

	// Update cache
	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Updating URL in cache")
		err := s.cache.Set(ctx, urlRecord)
		if err != nil {
			return nil, err
		}
	}

	log.Info().
		Str("short", short).
		Str("original_url", urlRecord.Original).
		Time("expires_at", urlRecord.ExpiresAt).
		Int64("clicks", urlRecord.Clicks).
		Msg("URL retrieved by short code")

	return urlRecord, nil
}

// GetByOriginal retrieves a URL by its original URL
func (s *URLService) GetByOriginal(ctx context.Context, original string) (*models.URL, error) {
	log.Debug().Str("original_url", original).Msg("Getting URL by original URL")

	// Try to get from cache first
	if s.cache != nil {
		log.Debug().Str("original_url", original).Msg("Checking cache for URL")
		urlData, err := s.cache.GetByOriginal(ctx, original)
		if err == nil {
			log.Debug().Str("original_url", original).Msg("URL found in cache")
			return urlData, nil
		} else if !errors.Is(err, ErrURLNotFound) {
			log.Error().Err(err).Str("original_url", original).Msg("Cache error when getting URL by original URL")
		} else {
			log.Debug().Str("original_url", original).Msg("URL not found in cache, checking database")
		}
	}

	// Get from database
	log.Debug().Str("original_url", original).Msg("Getting URL from database")
	urlRecord, err := s.db.GetByOriginal(ctx, original)
	if err != nil {
		if errors.Is(err, ErrURLNotFound) {
			log.Debug().Str("original_url", original).Msg("URL not found in database")
		} else {
			log.Error().Err(err).Str("original_url", original).Msg("Database error when getting URL by original URL")
		}
		return nil, err
	}

	// Update cache
	if s.cache != nil {
		log.Debug().Str("original_url", original).Msg("Updating URL in cache")
		err := s.cache.Set(ctx, urlRecord)
		if err != nil {
			return nil, err
		}
	}

	log.Info().
		Str("original_url", original).
		Str("short", urlRecord.Short).
		Time("expires_at", urlRecord.ExpiresAt).
		Int64("clicks", urlRecord.Clicks).
		Msg("URL retrieved by original URL")

	return urlRecord, nil
}

// GetByCreator retrieves URLs by their creator reference
func (s *URLService) GetByCreator(ctx context.Context, creatorReference string) ([]*models.URL, error) {
	log.Debug().Str("creator_reference", creatorReference).Msg("Getting URLs by creator reference")

	// Get from database
	log.Debug().Str("creator_reference", creatorReference).Msg("Getting URLs from database")
	urlRecords, err := s.db.GetByCreator(ctx, creatorReference)
	if err != nil {
		log.Error().Err(err).Str("creator_reference", creatorReference).Msg("Database error when getting URLs by creator reference")
		return nil, err
	}

	if len(urlRecords) == 0 {
		log.Debug().Str("creator_reference", creatorReference).Msg("No URLs found for creator reference")
	} else {
		log.Info().
			Str("creator_reference", creatorReference).
			Int("count", len(urlRecords)).
			Msg("URLs retrieved by creator reference")
	}

	return urlRecords, nil
}

// IncrementClicks increments the click count for a URL
func (s *URLService) IncrementClicks(ctx context.Context, short string) error {
	log.Debug().Str("short", short).Msg("Incrementing click count")

	// Update database
	if err := s.db.IncrementClicks(ctx, short); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to increment click count in database")
		return err
	}

	// Update cache if it exists
	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Updating click count in cache")
		if err := s.cache.IncrementClicks(ctx, short); err != nil {
			// We don't return cache errors as the database update was successful
			log.Warn().Err(err).Str("short", short).Msg("Failed to increment click count in cache")
		}
	}

	log.Debug().Str("short", short).Msg("Click count incremented successfully")
	return nil
}

// Delete removes a URL
func (s *URLService) Delete(ctx context.Context, short string) error {
	log.Debug().Str("short", short).Msg("Deleting URL")

	// Get URL before deleting to log history
	url, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for deletion")
		return err
	}

	// Log URL deletion history
	if err := s.db.LogURLHistory(ctx, url.ID, short, "delete", url, nil, ""); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to log URL deletion history")
		// Continue with deletion even if logging fails
	}

	// Delete from database
	if err := s.db.Delete(ctx, short); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to delete URL from database")
		return err
	}

	// Delete from cache
	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Deleting URL from cache")
		if err := s.cache.Delete(ctx, short); err != nil {
			// We don't return cache errors as the database delete was successful
			log.Warn().Err(err).Str("short", short).Msg("Failed to delete URL from cache")
		}
	}

	log.Info().Str("short", short).Msg("URL deleted successfully")
	return nil
}

// DeleteWithCreator removes a URL if the creator_reference matches
func (s *URLService) DeleteWithCreator(ctx context.Context, short string, creatorReference string) error {
	log.Debug().Str("short", short).Str("creator_reference", creatorReference).Msg("Deleting URL with creator reference check")

	// Get URL before deleting to log history
	url, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for deletion")
		return err
	}

	// Log URL deletion history
	if err := s.db.LogURLHistory(ctx, url.ID, short, "delete", url, nil, creatorReference); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to log URL deletion history")
		// Continue with deletion even if logging fails
	}

	// Delete from database with creator reference check
	if err := s.db.DeleteWithCreator(ctx, short, creatorReference); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to delete URL from database")
		return err
	}

	// Delete from cache
	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Deleting URL from cache")
		if err := s.cache.Delete(ctx, short); err != nil {
			// We don't return cache errors as the database delete was successful
			log.Warn().Err(err).Str("short", short).Msg("Failed to delete URL from cache")
		}
	}

	log.Info().Str("short", short).Str("creator_reference", creatorReference).Msg("URL deleted successfully")
	return nil
}

// UpdateURL updates an existing URL
func (s *URLService) UpdateURL(ctx context.Context, short string, title, originalURL string, expireAfter time.Duration) (*models.URL, error) {
	log.Debug().
		Str("short", short).
		Str("title", title).
		Str("original_url", originalURL).
		Dur("expire_after", expireAfter).
		Msg("Updating URL")

	// Get existing URL
	existingURL, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for update")
		return nil, err
	}

	// Validate URL if changed
	if originalURL != existingURL.Original {
		if _, err := url.ParseRequestURI(originalURL); err != nil {
			log.Error().Err(err).Str("url", originalURL).Msg("Invalid URL format")
			return nil, ErrInvalidURL
		}
	}

	// Create updated URL
	updatedURL := &models.URL{
		ID:               existingURL.ID,
		Original:         originalURL,
		Short:            short,
		Title:            title,
		CreatedAt:        existingURL.CreatedAt,
		Clicks:           existingURL.Clicks,
		CreatorReference: existingURL.CreatorReference,
	}

	// Set expiration time if provided
	if expireAfter > 0 {
		updatedURL.ExpiresAt = time.Now().Add(expireAfter)
		log.Debug().Time("expires_at", updatedURL.ExpiresAt).Msg("Setting URL expiration time")
	} else {
		updatedURL.ExpiresAt = existingURL.ExpiresAt
	}

	// Log URL update history
	if err := s.db.LogURLHistory(ctx, existingURL.ID, short, "update", existingURL, updatedURL, ""); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to log URL update history")
		// Continue with update even if logging fails
	}

	// Update URL in database
	if err := s.db.UpdateURL(ctx, short, updatedURL); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to update URL in database")
		return nil, err
	}

	// Update cache
	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Updating URL in cache")
		if err := s.cache.Set(ctx, updatedURL); err != nil {
			log.Warn().Err(err).Str("short", short).Msg("Failed to update URL in cache")
			// Continue even if cache update fails
		}
	}

	log.Info().
		Str("short", short).
		Str("original_url", updatedURL.Original).
		Str("title", updatedURL.Title).
		Time("expires_at", updatedURL.ExpiresAt).
		Msg("URL updated successfully")

	return updatedURL, nil
}

// UpdateURLWithCreator updates an existing URL if the creator_reference matches
func (s *URLService) UpdateURLWithCreator(ctx context.Context, short string, title, originalURL string, expireAfter time.Duration, creatorReference string) (*models.URL, error) {
	log.Debug().
		Str("short", short).
		Str("title", title).
		Str("original_url", originalURL).
		Dur("expire_after", expireAfter).
		Str("creator_reference", creatorReference).
		Msg("Updating URL with creator reference check")

	// Get existing URL
	existingURL, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for update")
		return nil, err
	}

	// Validate URL if changed
	if originalURL != existingURL.Original {
		if _, err := url.ParseRequestURI(originalURL); err != nil {
			log.Error().Err(err).Str("url", originalURL).Msg("Invalid URL format")
			return nil, ErrInvalidURL
		}
	}

	// Create updated URL
	updatedURL := &models.URL{
		ID:               existingURL.ID,
		Original:         originalURL,
		Short:            short,
		Title:            title,
		CreatedAt:        existingURL.CreatedAt,
		Clicks:           existingURL.Clicks,
		CreatorReference: existingURL.CreatorReference,
	}

	// Set expiration time if provided
	if expireAfter > 0 {
		updatedURL.ExpiresAt = time.Now().Add(expireAfter)
		log.Debug().Time("expires_at", updatedURL.ExpiresAt).Msg("Setting URL expiration time")
	} else {
		updatedURL.ExpiresAt = existingURL.ExpiresAt
	}

	// Log URL update history
	if err := s.db.LogURLHistory(ctx, existingURL.ID, short, "update", existingURL, updatedURL, creatorReference); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to log URL update history")
		// Continue with update even if logging fails
	}

	// Update URL in database with creator reference check
	if err := s.db.UpdateURLWithCreator(ctx, short, updatedURL, creatorReference); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to update URL in database")
		return nil, err
	}

	// Update cache
	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Updating URL in cache")
		if err := s.cache.Set(ctx, updatedURL); err != nil {
			log.Warn().Err(err).Str("short", short).Msg("Failed to update URL in cache")
			// Continue even if cache update fails
		}
	}

	log.Info().
		Str("short", short).
		Str("original_url", updatedURL.Original).
		Str("title", updatedURL.Title).
		Time("expires_at", updatedURL.ExpiresAt).
		Str("creator_reference", creatorReference).
		Msg("URL updated successfully")

	return updatedURL, nil
}

// generateShortURL generates a random short URL
func (s *URLService) generateShortURL(length int) (string, error) {
	log.Debug().Int("length", length).Msg("Generating random short URL")

	for i := 0; i < 5; i++ { // Try up to 5 times
		log.Debug().Int("attempt", i+1).Msg("Attempting to generate short URL")

		// Generate random bytes
		b := make([]byte, length)
		_, err := rand.Read(b)
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate random bytes")
			return "", err
		}

		// Encode to base64 and clean up
		encoded := base64.URLEncoding.EncodeToString(b)
		// Remove padding characters and take only the first 'length' characters
		short := strings.ReplaceAll(encoded, "=", "")[:length]

		log.Debug().Str("short", short).Msg("Generated short code, checking if it exists")

		// Check if it already exists
		_, err = s.GetByShort(context.Background(), short)
		if errors.Is(err, ErrURLNotFound) {
			// This short URL is available
			log.Debug().Str("short", short).Msg("Short code is available")
			return short, nil
		} else if err != nil && !errors.Is(err, ErrURLNotFound) {
			log.Error().Err(err).Str("short", short).Msg("Error checking if short code exists")
		} else {
			log.Debug().Str("short", short).Msg("Short code already exists, trying again")
		}
	}

	log.Error().Msg("Failed to generate unique short URL after 5 attempts")
	return "", ErrURLExists
}

// RecordClick records click analytics data
func (s *URLService) RecordClick(ctx context.Context, short string, ip, location, browser, device string) error {
	log.Debug().
		Str("short", short).
		Str("ip", ip).
		Str("location", location).
		Str("browser", browser).
		Str("device", device).
		Msg("Recording click analytics")

	// Check if there's a recent click from the same visitor
	hasRecentClick, err := s.db.HasRecentClick(ctx, short, ip, browser, device)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to check for recent clicks")
		return err
	}

	if hasRecentClick {
		log.Debug().
			Str("short", short).
			Str("ip", ip).
			Str("browser", browser).
			Str("device", device).
			Msg("Recent click from the same visitor found, skipping recording")
		return ErrRecentClick
	}

	// Get URL to get the ID
	shortURL, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for recording click")
		return err
	}

	// Create click record
	click := models.NewClick(shortURL.ID, short, ip, location, browser, device)

	// Store click data
	if err := s.db.StoreClick(ctx, click); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to store click analytics")
		return err
	}

	log.Info().
		Str("short", short).
		Str("ip", ip).
		Msg("Click analytics recorded successfully")

	return nil
}

// GetClicksByShort retrieves click analytics data for a URL
func (s *URLService) GetClicksByShort(ctx context.Context, short string) ([]*models.Click, error) {
	log.Debug().Str("short", short).Msg("Getting click analytics data")

	clicks, err := s.db.GetClicksByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get click analytics data")
		return nil, err
	}

	log.Info().
		Str("short", short).
		Int("count", len(clicks)).
		Msg("Click analytics data retrieved successfully")

	return clicks, nil
}

// GetClickAnalytics retrieves aggregated click analytics data for a URL
func (s *URLService) GetClickAnalytics(ctx context.Context, short string) (map[string]interface{}, error) {
	log.Debug().Str("short", short).Msg("Getting aggregated click analytics data")

	analytics, err := s.db.GetClickAnalytics(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get aggregated click analytics data")
		return nil, err
	}

	log.Info().
		Str("short", short).
		Interface("analytics", analytics).
		Msg("Aggregated click analytics data retrieved successfully")

	return analytics, nil
}
