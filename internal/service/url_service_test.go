package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fransfilastap/urlshortener/internal/domain"
	"github.com/fransfilastap/urlshortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockURLRepository struct {
	mock.Mock
}

func (m *MockURLRepository) Create(ctx context.Context, url *domain.URL) (*domain.URL, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLRepository) GetByShort(ctx context.Context, short string) (*domain.URL, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLRepository) GetByOriginal(ctx context.Context, original string) (*domain.URL, error) {
	args := m.Called(ctx, original)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLRepository) IncrementClicks(ctx context.Context, short string) error {
	args := m.Called(ctx, short)
	return args.Error(0)
}

func (m *MockURLRepository) Delete(ctx context.Context, short string) error {
	args := m.Called(ctx, short)
	return args.Error(0)
}

func (m *MockURLRepository) StoreClick(ctx context.Context, click *domain.Click) error {
	args := m.Called(ctx, click)
	return args.Error(0)
}

func (m *MockURLRepository) GetClicksByShort(ctx context.Context, short string) ([]*domain.Click, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Click), args.Error(1)
}

func (m *MockURLRepository) GetClickAnalytics(ctx context.Context, short string) (map[string]interface{}, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockURLRepository) HasRecentClick(ctx context.Context, short string, ip string, browser string, device string) (bool, error) {
	args := m.Called(ctx, short, ip, browser, device)
	return args.Bool(0), args.Error(1)
}

func (m *MockURLRepository) UpdateURL(ctx context.Context, short string, url *domain.URL) error {
	args := m.Called(ctx, short, url)
	return args.Error(0)
}

func (m *MockURLRepository) LogURLHistory(ctx context.Context, urlID int64, short string, action string, oldValue, newValue interface{}, modifiedBy string) error {
	args := m.Called(ctx, urlID, short, action, oldValue, newValue, modifiedBy)
	return args.Error(0)
}

func (m *MockURLRepository) DeleteWithCreator(ctx context.Context, short string, creatorReference string) error {
	args := m.Called(ctx, short, creatorReference)
	return args.Error(0)
}

func (m *MockURLRepository) GetByCreator(ctx context.Context, creatorReference string) ([]*domain.URL, error) {
	args := m.Called(ctx, creatorReference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.URL), args.Error(1)
}

func (m *MockURLRepository) UpdateURLWithCreator(ctx context.Context, short string, url *domain.URL, creatorReference string) error {
	args := m.Called(ctx, short, url, creatorReference)
	return args.Error(0)
}

type MockCacheRepository struct {
	mock.Mock
}

var _ repository.CacheRepositoryInterface = (*MockCacheRepository)(nil)

func (m *MockCacheRepository) Set(ctx context.Context, url *domain.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockCacheRepository) GetByShort(ctx context.Context, short string) (*domain.URL, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockCacheRepository) GetByOriginal(ctx context.Context, original string) (*domain.URL, error) {
	args := m.Called(ctx, original)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockCacheRepository) IncrementClicks(ctx context.Context, short string) error {
	args := m.Called(ctx, short)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, short string) error {
	args := m.Called(ctx, short)
	return args.Error(0)
}

func (m *MockCacheRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestCreateShortURL(t *testing.T) {
	mockRepo := new(MockURLRepository)
	mockCache := new(MockCacheRepository)
	svc := NewURLService(mockRepo, mockCache)
	ctx := context.Background()

	t.Run("CreateNewURL", func(t *testing.T) {
		originalURL := "https://example.com"
		customShort := "custom"
		expireAfter := time.Hour

		mockRepo.On("GetByShort", ctx, customShort).Return(nil, domain.ErrURLNotFound)
		mockRepo.On("GetByOriginal", ctx, originalURL).Maybe().Return(nil, domain.ErrURLNotFound)
		mockRepo.On("GetByShort", ctx, mock.MatchedBy(func(s string) bool { return s != customShort })).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByOriginal", ctx, originalURL).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByShort", ctx, customShort).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByShort", ctx, mock.MatchedBy(func(s string) bool { return s != customShort })).Maybe().Return(nil, domain.ErrURLNotFound)

		dummyURL := &domain.URL{
			ID:               1,
			Original:         originalURL,
			Short:            customShort,
			Title:            "Test Title",
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(expireAfter),
			Clicks:           0,
			CreatorReference: "test-user",
		}
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).Return(dummyURL, nil)
		mockCache.On("Set", ctx, mock.AnythingOfType("*domain.URL")).Return(nil)

		url, err := svc.CreateShortURL(ctx, originalURL, customShort, "Test Title", expireAfter, "test-user")

		assert.NoError(t, err)
		assert.NotNil(t, url)
		assert.Equal(t, originalURL, url.Original)
		assert.Equal(t, customShort, url.Short)
		assert.WithinDuration(t, time.Now().Add(expireAfter), url.ExpiresAt, time.Second)
		assert.Equal(t, int64(0), url.Clicks)

		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("URLAlreadyExists", func(t *testing.T) {
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		svc := NewURLService(mockRepo, mockCache)

		originalURL := "https://example.com"
		existingURL := &domain.URL{
			Original:  originalURL,
			Short:     "existing",
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    5,
		}

		mockRepo.On("GetByShort", ctx, mock.AnythingOfType("string")).Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByShort", ctx, mock.AnythingOfType("string")).Return(nil, domain.ErrURLNotFound)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).Return(existingURL, nil)
		mockRepo.On("GetByOriginal", ctx, originalURL).Maybe().Return(existingURL, nil)
		mockCache.On("GetByOriginal", ctx, originalURL).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("Set", ctx, mock.AnythingOfType("*domain.URL")).Maybe().Return(nil)

		url, err := svc.CreateShortURL(ctx, originalURL, "", "", time.Duration(0), "")

		assert.NoError(t, err)
		assert.Equal(t, existingURL, url)

		mockRepo.AssertExpectations(t)
	})

	t.Run("InvalidURL", func(t *testing.T) {
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		svc := NewURLService(mockRepo, mockCache)

		mockRepo.On("GetByShort", ctx, mock.AnythingOfType("string")).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByShort", ctx, mock.AnythingOfType("string")).Maybe().Return(nil, domain.ErrURLNotFound)
		mockRepo.On("GetByOriginal", ctx, mock.AnythingOfType("string")).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByOriginal", ctx, mock.AnythingOfType("string")).Maybe().Return(nil, domain.ErrURLNotFound)

		dummyURL := &domain.URL{
			ID:        1,
			Original:  "invalid-url",
			Short:     "dummy",
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    0,
		}
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).Maybe().Return(dummyURL, nil)
		mockCache.On("Set", ctx, mock.AnythingOfType("*domain.URL")).Maybe().Return(nil)

		url, err := svc.CreateShortURL(ctx, "invalid-url", "", "", time.Duration(0), "")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrInvalidURL), "expected ErrInvalidURL, got: %v", err)
		assert.Nil(t, url)
	})

	t.Run("CustomShortExists", func(t *testing.T) {
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		svc := NewURLService(mockRepo, mockCache)

		originalURL := "https://example.com"
		customShort := "existing"
		existingURL := &domain.URL{
			Original:  "https://another-example.com",
			Short:     customShort,
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    5,
		}

		mockRepo.On("GetByShort", ctx, customShort).Return(existingURL, nil)
		mockRepo.On("GetByOriginal", ctx, originalURL).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByOriginal", ctx, originalURL).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByShort", ctx, customShort).Maybe().Return(nil, domain.ErrURLNotFound)
		mockRepo.On("GetByShort", ctx, mock.MatchedBy(func(s string) bool { return s != customShort })).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("GetByShort", ctx, mock.MatchedBy(func(s string) bool { return s != customShort })).Maybe().Return(nil, domain.ErrURLNotFound)
		mockCache.On("Set", ctx, mock.AnythingOfType("*domain.URL")).Maybe().Return(nil)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).Maybe().Return(existingURL, nil)

		url, err := svc.CreateShortURL(ctx, originalURL, customShort, "", time.Duration(0), "")

		assert.Error(t, err)
		assert.Equal(t, domain.ErrURLExists, err)
		assert.Nil(t, url)

		mockRepo.AssertExpectations(t)
	})
}

func TestGetByShort(t *testing.T) {
	mockRepo := new(MockURLRepository)
	mockCache := new(MockCacheRepository)
	svc := NewURLService(mockRepo, mockCache)
	ctx := context.Background()

	t.Run("GetFromCache", func(t *testing.T) {
		short := "abc123"
		url := &domain.URL{
			Original:  "https://example.com",
			Short:     short,
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    5,
		}

		mockCache.On("GetByShort", ctx, short).Return(url, nil)

		result, err := svc.GetByShort(ctx, short)

		assert.NoError(t, err)
		assert.Equal(t, url, result)

		mockCache.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "GetByShort")
	})

	t.Run("GetFromDatabase", func(t *testing.T) {
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		svc := NewURLService(mockRepo, mockCache)

		short := "abc123"
		url := &domain.URL{
			Original:  "https://example.com",
			Short:     short,
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    5,
		}

		mockCache.On("GetByShort", ctx, short).Return(nil, domain.ErrURLNotFound)
		mockRepo.On("GetByShort", ctx, short).Return(url, nil)
		mockCache.On("Set", ctx, mock.AnythingOfType("*domain.URL")).Return(nil)

		result, err := svc.GetByShort(ctx, short)

		assert.NoError(t, err)
		assert.Equal(t, url.Original, result.Original)
		assert.Equal(t, url.Short, result.Short)
		assert.Equal(t, url.Clicks, result.Clicks)
		assert.WithinDuration(t, url.CreatedAt, result.CreatedAt, time.Second)

		mockCache.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("URLNotFound", func(t *testing.T) {
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		svc := NewURLService(mockRepo, mockCache)

		short := "notfound"

		mockCache.On("GetByShort", ctx, short).Return(nil, domain.ErrURLNotFound)
		mockRepo.On("GetByShort", ctx, short).Return(nil, domain.ErrURLNotFound)

		result, err := svc.GetByShort(ctx, short)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrURLNotFound, err)
		assert.Nil(t, result)

		mockCache.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}

func TestIncrementClicks(t *testing.T) {
	mockRepo := new(MockURLRepository)
	mockCache := new(MockCacheRepository)
	svc := NewURLService(mockRepo, mockCache)
	ctx := context.Background()

	t.Run("IncrementClicks", func(t *testing.T) {
		short := "abc123"

		mockRepo.On("IncrementClicks", ctx, short).Return(nil)
		mockCache.On("IncrementClicks", ctx, short).Return(nil)

		err := svc.IncrementClicks(ctx, short)

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("DatabaseError", func(t *testing.T) {
		short := "error123"
		dbErr := assert.AnError

		mockRepo.On("IncrementClicks", ctx, short).Return(dbErr)

		err := svc.IncrementClicks(ctx, short)

		assert.Error(t, err)
		assert.Equal(t, dbErr, err)

		mockRepo.AssertExpectations(t)
		mockCache.AssertNotCalled(t, "IncrementClicks")
	})
}