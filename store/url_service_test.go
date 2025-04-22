package store

import (
	"context"
	"testing"
	"time"

	"github.com/fransfilastap/urlshortener/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockURLRepository is a mock implementation of the URL repository
type MockURLRepository struct {
	mock.Mock
}

func (m *MockURLRepository) Create(ctx context.Context, url *models.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockURLRepository) GetByShort(ctx context.Context, short string) (*models.URL, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockURLRepository) GetByOriginal(ctx context.Context, original string) (*models.URL, error) {
	args := m.Called(ctx, original)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockURLRepository) IncrementClicks(ctx context.Context, short string) error {
	args := m.Called(ctx, short)
	return args.Error(0)
}

func (m *MockURLRepository) Delete(ctx context.Context, short string) error {
	args := m.Called(ctx, short)
	return args.Error(0)
}

func (m *MockURLRepository) StoreClick(ctx context.Context, click *models.Click) error {
	args := m.Called(ctx, click)
	return args.Error(0)
}

func (m *MockURLRepository) GetClicksByShort(ctx context.Context, short string) ([]*models.Click, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Click), args.Error(1)
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

func (m *MockURLRepository) UpdateURL(ctx context.Context, short string, url *models.URL) error {
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

func (m *MockURLRepository) GetByCreator(ctx context.Context, creatorReference string) ([]*models.URL, error) {
	args := m.Called(ctx, creatorReference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.URL), args.Error(1)
}

func (m *MockURLRepository) UpdateURLWithCreator(ctx context.Context, short string, url *models.URL, creatorReference string) error {
	args := m.Called(ctx, short, url, creatorReference)
	return args.Error(0)
}

// MockCacheRepository is a mock implementation of the cache repository
type MockCacheRepository struct {
	mock.Mock
}

// Ensure MockCacheRepository implements CacheRepositoryInterface
var _ CacheRepositoryInterface = (*MockCacheRepository)(nil)

func (m *MockCacheRepository) Set(ctx context.Context, url *models.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockCacheRepository) GetByShort(ctx context.Context, short string) (*models.URL, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockCacheRepository) GetByOriginal(ctx context.Context, original string) (*models.URL, error) {
	args := m.Called(ctx, original)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
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
	// Setup
	mockRepo := new(MockURLRepository)
	mockCache := new(MockCacheRepository)
	service := NewURLService(mockRepo, mockCache)
	ctx := context.Background()

	// Test case 1: Create a new short URL
	t.Run("CreateNewURL", func(t *testing.T) {
		// Setup expectations
		originalURL := "https://example.com"
		customShort := "custom"
		expireAfter := time.Hour

		// Mock repository behavior
		mockRepo.On("GetByOriginal", ctx, originalURL).Return(nil, ErrURLNotFound)
		mockRepo.On("GetByShort", ctx, customShort).Return(nil, ErrURLNotFound)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.URL")).Return(nil)
		mockCache.On("GetByOriginal", ctx, originalURL).Return(nil, ErrURLNotFound)
		mockCache.On("GetByShort", ctx, customShort).Return(nil, ErrURLNotFound)
		mockCache.On("Set", ctx, mock.AnythingOfType("*models.URL")).Return(nil)

		// Call the service
		url, err := service.CreateShortURL(ctx, originalURL, customShort, "Test Title", expireAfter, "test-user")

		// Assertions
		assert.NoError(t, err)
		assert.NotNil(t, url)
		assert.Equal(t, originalURL, url.Original)
		assert.Equal(t, customShort, url.Short)
		assert.WithinDuration(t, time.Now().Add(expireAfter), url.ExpiresAt, time.Second)
		assert.Equal(t, int64(0), url.Clicks)

		// Verify mocks
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	// Test case 2: URL already exists
	t.Run("URLAlreadyExists", func(t *testing.T) {
		// Create new mocks for this test to avoid interference
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		service := NewURLService(mockRepo, mockCache)

		// Setup expectations
		originalURL := "https://example.com"
		existingURL := &models.URL{
			Original:  originalURL,
			Short:     "existing",
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    5,
		}

		// Mock repository behavior
		mockRepo.On("GetByOriginal", ctx, originalURL).Return(existingURL, nil)
		mockCache.On("GetByOriginal", ctx, originalURL).Return(nil, ErrURLNotFound)
		mockCache.On("Set", ctx, mock.AnythingOfType("*models.URL")).Return(nil)

		// Call the service
		url, err := service.CreateShortURL(ctx, originalURL, "", "", time.Duration(0), "")

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, existingURL, url)

		// Verify mocks
		mockRepo.AssertExpectations(t)
	})

	// Test case 3: Invalid URL
	t.Run("InvalidURL", func(t *testing.T) {
		// Call the service with an invalid URL
		url, err := service.CreateShortURL(ctx, "invalid-url", "", "", time.Duration(0), "")

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidURL, err)
		assert.Nil(t, url)
	})

	// Test case 4: Custom short code already exists
	t.Run("CustomShortExists", func(t *testing.T) {
		// Create new mocks for this test to avoid interference
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		service := NewURLService(mockRepo, mockCache)

		// Setup expectations
		originalURL := "https://example.com"
		customShort := "existing"
		existingURL := &models.URL{
			Original:  "https://another-example.com",
			Short:     customShort,
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    5,
		}

		// Mock repository behavior
		mockRepo.On("GetByOriginal", ctx, originalURL).Return(nil, ErrURLNotFound)
		mockCache.On("GetByOriginal", ctx, originalURL).Return(nil, ErrURLNotFound)
		mockRepo.On("GetByShort", ctx, customShort).Return(existingURL, nil)
		mockCache.On("GetByShort", ctx, customShort).Return(nil, ErrURLNotFound)
		mockCache.On("Set", ctx, mock.AnythingOfType("*models.URL")).Return(nil)

		// Call the service
		url, err := service.CreateShortURL(ctx, originalURL, customShort, "", time.Duration(0), "")

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrURLExists, err)
		assert.Nil(t, url)

		// Verify mocks
		mockRepo.AssertExpectations(t)
	})
}

func TestGetByShort(t *testing.T) {
	// Setup
	mockRepo := new(MockURLRepository)
	mockCache := new(MockCacheRepository)
	service := NewURLService(mockRepo, mockCache)
	ctx := context.Background()

	// Test case 1: Get URL from cache
	t.Run("GetFromCache", func(t *testing.T) {
		// Setup expectations
		short := "abc123"
		url := &models.URL{
			Original:  "https://example.com",
			Short:     short,
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    5,
		}

		// Mock cache behavior
		mockCache.On("GetByShort", ctx, short).Return(url, nil)

		// Call the service
		result, err := service.GetByShort(ctx, short)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, url, result)

		// Verify mocks
		mockCache.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "GetByShort")
	})

	// Test case 2: Get URL from database
	t.Run("GetFromDatabase", func(t *testing.T) {
		// Create new mocks for this test to avoid interference
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		service := NewURLService(mockRepo, mockCache)

		// Setup expectations
		short := "abc123"
		url := &models.URL{
			Original:  "https://example.com",
			Short:     short,
			CreatedAt: time.Now(),
			ExpiresAt: time.Time{},
			Clicks:    5,
		}

		// Mock behavior
		mockCache.On("GetByShort", ctx, short).Return(nil, ErrURLNotFound)
		mockRepo.On("GetByShort", ctx, short).Return(url, nil)
		mockCache.On("Set", ctx, mock.AnythingOfType("*models.URL")).Return(nil)

		// Call the service
		result, err := service.GetByShort(ctx, short)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, url.Original, result.Original)
		assert.Equal(t, url.Short, result.Short)
		assert.Equal(t, url.Clicks, result.Clicks)
		assert.WithinDuration(t, url.CreatedAt, result.CreatedAt, time.Second)

		// Verify mocks
		mockCache.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	// Test case 3: URL not found
	t.Run("URLNotFound", func(t *testing.T) {
		// Create new mocks for this test to avoid interference
		mockRepo := new(MockURLRepository)
		mockCache := new(MockCacheRepository)
		service := NewURLService(mockRepo, mockCache)

		// Setup expectations
		short := "notfound"

		// Mock behavior
		mockCache.On("GetByShort", ctx, short).Return(nil, ErrURLNotFound)
		mockRepo.On("GetByShort", ctx, short).Return(nil, ErrURLNotFound)

		// Call the service
		result, err := service.GetByShort(ctx, short)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrURLNotFound, err)
		assert.Nil(t, result)

		// Verify mocks
		mockCache.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}

func TestIncrementClicks(t *testing.T) {
	// Setup
	mockRepo := new(MockURLRepository)
	mockCache := new(MockCacheRepository)
	service := NewURLService(mockRepo, mockCache)
	ctx := context.Background()

	// Test case: Increment clicks
	t.Run("IncrementClicks", func(t *testing.T) {
		// Setup expectations
		short := "abc123"

		// Mock behavior
		mockRepo.On("IncrementClicks", ctx, short).Return(nil)
		mockCache.On("IncrementClicks", ctx, short).Return(nil)

		// Call the service
		err := service.IncrementClicks(ctx, short)

		// Assertions
		assert.NoError(t, err)

		// Verify mocks
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	// Test case: Database error
	t.Run("DatabaseError", func(t *testing.T) {
		// Setup expectations
		short := "error123"
		dbErr := assert.AnError

		// Mock behavior
		mockRepo.On("IncrementClicks", ctx, short).Return(dbErr)

		// Call the service
		err := service.IncrementClicks(ctx, short)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, dbErr, err)

		// Verify mocks
		mockRepo.AssertExpectations(t)
		mockCache.AssertNotCalled(t, "IncrementClicks")
	})
}
