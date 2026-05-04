package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/fransfilastap/urlshortener/internal/domain"
	"github.com/fransfilastap/urlshortener/internal/repository"
	"github.com/rs/zerolog/log"
)

type URLService struct {
	db    repository.URLRepository
cache repository.CacheRepositoryInterface
}

func NewURLService(db repository.URLRepository, cache repository.CacheRepositoryInterface) *URLService {
	return &URLService{
		db:    db,
		cache: cache,
	}
}

func (s *URLService) CreateShortURL(ctx context.Context, originalURL string, customShort string, title string, expireAfter time.Duration, creatorReference string) (*domain.URL, error) {
	log.Debug().
		Str("original_url", originalURL).
		Str("custom_short", customShort).
		Str("title", title).
		Dur("expire_after", expireAfter).
		Str("creator_reference", creatorReference).
		Msg("Creating short URL")

	if err := ValidateURL(originalURL); err != nil {
		log.Error().Err(err).Str("url", originalURL).Msg("Invalid URL format")
		return nil, err
	}

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
		if err := ValidateShortCode(short); err != nil {
			log.Error().Err(err).Str("custom_short", short).Msg("Invalid custom short code")
			return nil, err
		}
		_, err := s.GetByShort(ctx, short)
		if err == nil {
			log.Error().Str("custom_short", short).Msg("Custom short code already in use")
			return nil, domain.ErrURLExists
		} else if !errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("custom_short", short).Msg("Error checking if custom short code exists")
			return nil, err
		}
	}

	var expiresAt time.Time
	if expireAfter > 0 {
		expiresAt = time.Now().Add(expireAfter)
		log.Debug().Time("expires_at", expiresAt).Msg("Setting URL expiration time")
	}

	newURL := domain.NewURL(originalURL, short, title, expiresAt, creatorReference)

	log.Debug().Str("short", short).Msg("Saving URL to database")
	createdURL, err := s.db.Create(ctx, newURL)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to save URL to database")
		return nil, err
	}

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

func (s *URLService) GetByShort(ctx context.Context, short string) (*domain.URL, error) {
	log.Debug().Str("short", short).Msg("Getting URL by short code")

	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Checking cache for URL")
		foundURL, err := s.cache.GetByShort(ctx, short)
		if err == nil {
			log.Debug().Str("short", short).Msg("URL found in cache")
			return foundURL, nil
		} else if !errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("short", short).Msg("Cache error when getting URL by short code")
		} else {
			log.Debug().Str("short", short).Msg("URL not found in cache, checking database")
		}
	}

	log.Debug().Str("short", short).Msg("Getting URL from database")
	urlRecord, err := s.db.GetByShort(ctx, short)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			log.Debug().Str("short", short).Msg("URL not found in database")
		} else {
			log.Error().Err(err).Str("short", short).Msg("Database error when getting URL by short code")
		}
		return nil, err
	}

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

func (s *URLService) GetByOriginal(ctx context.Context, original string) (*domain.URL, error) {
	log.Debug().Str("original_url", original).Msg("Getting URL by original URL")

	if s.cache != nil {
		log.Debug().Str("original_url", original).Msg("Checking cache for URL")
		urlData, err := s.cache.GetByOriginal(ctx, original)
		if err == nil {
			log.Debug().Str("original_url", original).Msg("URL found in cache")
			return urlData, nil
		} else if !errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("original_url", original).Msg("Cache error when getting URL by original URL")
		} else {
			log.Debug().Str("original_url", original).Msg("URL not found in cache, checking database")
		}
	}

	log.Debug().Str("original_url", original).Msg("Getting URL from database")
	urlRecord, err := s.db.GetByOriginal(ctx, original)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			log.Debug().Str("original_url", original).Msg("URL not found in database")
		} else {
			log.Error().Err(err).Str("original_url", original).Msg("Database error when getting URL by original URL")
		}
		return nil, err
	}

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

func (s *URLService) GetByCreator(ctx context.Context, creatorReference string) ([]*domain.URL, error) {
	log.Debug().Str("creator_reference", creatorReference).Msg("Getting URLs by creator reference")

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

func (s *URLService) IncrementClicks(ctx context.Context, short string) error {
	log.Debug().Str("short", short).Msg("Incrementing click count")

	if err := s.db.IncrementClicks(ctx, short); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to increment click count in database")
		return err
	}

	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Updating click count in cache")
		if err := s.cache.IncrementClicks(ctx, short); err != nil {
			log.Warn().Err(err).Str("short", short).Msg("Failed to increment click count in cache")
		}
	}

	log.Debug().Str("short", short).Msg("Click count incremented successfully")
	return nil
}

func (s *URLService) Delete(ctx context.Context, short string) error {
	log.Debug().Str("short", short).Msg("Deleting URL")

	url, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for deletion")
		return err
	}

	if err := s.db.LogURLHistory(ctx, url.ID, short, "delete", url, nil, ""); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to log URL deletion history")
	}

	if err := s.db.Delete(ctx, short); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to delete URL from database")
		return err
	}

	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Deleting URL from cache")
		if err := s.cache.Delete(ctx, short); err != nil {
			log.Warn().Err(err).Str("short", short).Msg("Failed to delete URL from cache")
		}
	}

	log.Info().Str("short", short).Msg("URL deleted successfully")
	return nil
}

func (s *URLService) DeleteWithCreator(ctx context.Context, short string, creatorReference string) error {
	log.Debug().Str("short", short).Str("creator_reference", creatorReference).Msg("Deleting URL with creator reference check")

	url, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for deletion")
		return err
	}

	if err := s.db.LogURLHistory(ctx, url.ID, short, "delete", url, nil, creatorReference); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to log URL deletion history")
	}

	if err := s.db.DeleteWithCreator(ctx, short, creatorReference); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to delete URL from database")
		return err
	}

	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Deleting URL from cache")
		if err := s.cache.Delete(ctx, short); err != nil {
			log.Warn().Err(err).Str("short", short).Msg("Failed to delete URL from cache")
		}
	}

	log.Info().Str("short", short).Str("creator_reference", creatorReference).Msg("URL deleted successfully")
	return nil
}

func (s *URLService) UpdateURL(ctx context.Context, short string, title, originalURL string, expireAfter time.Duration) (*domain.URL, error) {
	log.Debug().
		Str("short", short).
		Str("title", title).
		Str("original_url", originalURL).
		Dur("expire_after", expireAfter).
		Msg("Updating URL")

	existingURL, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for update")
		return nil, err
	}

	if originalURL != existingURL.Original {
		if err := ValidateURL(originalURL); err != nil {
			log.Error().Err(err).Str("url", originalURL).Msg("Invalid URL format")
			return nil, err
		}
	}

	updatedURL := &domain.URL{
		ID:               existingURL.ID,
		Original:         originalURL,
		Short:            short,
		Title:            title,
		CreatedAt:        existingURL.CreatedAt,
		Clicks:           existingURL.Clicks,
		CreatorReference: existingURL.CreatorReference,
	}

	if expireAfter > 0 {
		updatedURL.ExpiresAt = time.Now().Add(expireAfter)
		log.Debug().Time("expires_at", updatedURL.ExpiresAt).Msg("Setting URL expiration time")
	} else {
		updatedURL.ExpiresAt = existingURL.ExpiresAt
	}

	if err := s.db.LogURLHistory(ctx, existingURL.ID, short, "update", existingURL, updatedURL, ""); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to log URL update history")
	}

	if err := s.db.UpdateURL(ctx, short, updatedURL); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to update URL in database")
		return nil, err
	}

	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Updating URL in cache")
		if err := s.cache.Set(ctx, updatedURL); err != nil {
			log.Warn().Err(err).Str("short", short).Msg("Failed to update URL in cache")
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

func (s *URLService) UpdateURLWithCreator(ctx context.Context, short string, title, originalURL string, expireAfter time.Duration, creatorReference string) (*domain.URL, error) {
	log.Debug().
		Str("short", short).
		Str("title", title).
		Str("original_url", originalURL).
		Dur("expire_after", expireAfter).
		Str("creator_reference", creatorReference).
		Msg("Updating URL with creator reference check")

	existingURL, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for update")
		return nil, err
	}

	if originalURL != existingURL.Original {
		if err := ValidateURL(originalURL); err != nil {
			log.Error().Err(err).Str("url", originalURL).Msg("Invalid URL format")
			return nil, err
		}
	}

	updatedURL := &domain.URL{
		ID:               existingURL.ID,
		Original:         originalURL,
		Short:            short,
		Title:            title,
		CreatedAt:        existingURL.CreatedAt,
		Clicks:           existingURL.Clicks,
		CreatorReference: existingURL.CreatorReference,
	}

	if expireAfter > 0 {
		updatedURL.ExpiresAt = time.Now().Add(expireAfter)
		log.Debug().Time("expires_at", updatedURL.ExpiresAt).Msg("Setting URL expiration time")
	} else {
		updatedURL.ExpiresAt = existingURL.ExpiresAt
	}

	if err := s.db.LogURLHistory(ctx, existingURL.ID, short, "update", existingURL, updatedURL, creatorReference); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to log URL update history")
	}

	if err := s.db.UpdateURLWithCreator(ctx, short, updatedURL, creatorReference); err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to update URL in database")
		return nil, err
	}

	if s.cache != nil {
		log.Debug().Str("short", short).Msg("Updating URL in cache")
		if err := s.cache.Set(ctx, updatedURL); err != nil {
			log.Warn().Err(err).Str("short", short).Msg("Failed to update URL in cache")
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

func (s *URLService) generateShortURL(length int) (string, error) {
	log.Debug().Int("length", length).Msg("Generating random short URL")

	for i := 0; i < 5; i++ {
		log.Debug().Int("attempt", i+1).Msg("Attempting to generate short URL")

		b := make([]byte, length)
		_, err := rand.Read(b)
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate random bytes")
			return "", err
		}

		encoded := base64.URLEncoding.EncodeToString(b)
		short := strings.ReplaceAll(encoded, "=", "")[:length]

		log.Debug().Str("short", short).Msg("Generated short code, checking if it exists")

		_, err = s.GetByShort(context.Background(), short)
		if errors.Is(err, domain.ErrURLNotFound) {
			log.Debug().Str("short", short).Msg("Short code is available")
			return short, nil
		} else if err != nil && !errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("short", short).Msg("Error checking if short code exists")
		} else {
			log.Debug().Str("short", short).Msg("Short code already exists, trying again")
		}
	}

	log.Error().Msg("Failed to generate unique short URL after 5 attempts")
	return "", domain.ErrURLExists
}

func (s *URLService) RecordClick(ctx context.Context, short string, ip, location, browser, device string) error {
	log.Debug().
		Str("short", short).
		Str("ip", ip).
		Str("location", location).
		Str("browser", browser).
		Str("device", device).
		Msg("Recording click analytics")

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
		return domain.ErrRecentClick
	}

	shortURL, err := s.GetByShort(ctx, short)
	if err != nil {
		log.Error().Err(err).Str("short", short).Msg("Failed to get URL for recording click")
		return err
	}

	click := domain.NewClick(shortURL.ID, short, ip, location, browser, device)

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

func (s *URLService) GetClicksByShort(ctx context.Context, short string) ([]*domain.Click, error) {
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