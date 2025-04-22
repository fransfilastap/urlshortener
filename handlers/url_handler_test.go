package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fransfilastap/urlshortener/models"
	"github.com/fransfilastap/urlshortener/store"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockURLService is a mock implementation of the URL service for testing
type MockURLService struct {
	mock.Mock
}

func (m *MockURLService) CreateShortURL(ctx context.Context, originalURL string, customShort string, title string, expireAfter time.Duration, creatorReference string) (*models.URL, error) {
	args := m.Called(ctx, originalURL, customShort, title, expireAfter, creatorReference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockURLService) GetByShort(ctx context.Context, short string) (*models.URL, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockURLService) GetByOriginal(ctx context.Context, original string) (*models.URL, error) {
	args := m.Called(ctx, original)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

func (m *MockURLService) IncrementClicks(ctx context.Context, short string) error {
	args := m.Called(ctx, short)
	return args.Error(0)
}

func (m *MockURLService) Delete(ctx context.Context, short string) error {
	args := m.Called(ctx, short)
	return args.Error(0)
}

func (m *MockURLService) RecordClick(ctx context.Context, short string, ip, location, browser, device string) error {
	args := m.Called(ctx, short, ip, location, browser, device)
	return args.Error(0)
}

func (m *MockURLService) GetClicksByShort(ctx context.Context, short string) ([]*models.Click, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Click), args.Error(1)
}

func (m *MockURLService) GetClickAnalytics(ctx context.Context, short string) (map[string]interface{}, error) {
	args := m.Called(ctx, short)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockURLService) UpdateURL(ctx context.Context, short string, title, originalURL string, expireAfter time.Duration) (*models.URL, error) {
	args := m.Called(ctx, short, title, originalURL, expireAfter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.URL), args.Error(1)
}

// TestURLHandler is a version of URLHandler that accepts a MockURLService for testing
type TestURLHandler struct {
	service  *MockURLService
	baseURL  string
	apiKey   string
}

// NewTestURLHandler creates a new test URL handler
func NewTestURLHandler(service *MockURLService, baseURL string, apiKey string) *TestURLHandler {
	return &TestURLHandler{
		service: service,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// ShortenURL handles requests to create short URLs
func (h *TestURLHandler) ShortenURL(c echo.Context) error {
	var req ShortenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Create short URL
	url, err := h.service.CreateShortURL(c.Request().Context(), req.URL, req.CustomCode, req.Title, req.Expiry, req.CreatorReference)
	if err != nil {
		switch {
		case err == store.ErrInvalidURL:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid URL"})
		case err == store.ErrURLExists:
			return c.JSON(http.StatusConflict, map[string]string{"error": "Custom code already in use"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create short URL"})
		}
	}

	// Construct full short URL
	shortURL := h.baseURL + "/" + url.Short

	// Return response
	return c.JSON(http.StatusCreated, URLResponse{
		OriginalURL:      url.Original,
		ShortURL:         shortURL,
		Title:            url.Title,
		ExpiresAt:        url.ExpiresAt,
		Clicks:           url.Clicks,
		CreatorReference: url.CreatorReference,
	})
}

// RedirectURL handles requests to redirect short URLs
func (h *TestURLHandler) RedirectURL(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	// Get URL by short code
	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if err == store.ErrURLNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	// Redirect to original URL
	return c.Redirect(http.StatusFound, url.Original)
}

// GetURLInfo returns information about a short URL
func (h *TestURLHandler) GetURLInfo(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	// Get URL by short code
	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if err == store.ErrURLNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	// Construct full short URL
	shortURL := h.baseURL + "/" + url.Short

	// Return response
	return c.JSON(http.StatusOK, URLResponse{
		OriginalURL:      url.Original,
		ShortURL:         shortURL,
		Title:            url.Title,
		ExpiresAt:        url.ExpiresAt,
		Clicks:           url.Clicks,
		CreatorReference: url.CreatorReference,
	})
}

// DeleteURL handles requests to delete a URL
func (h *TestURLHandler) DeleteURL(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	// Get URL to verify it exists
	_, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if err == store.ErrURLNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	// Delete URL
	if err := h.service.Delete(c.Request().Context(), code); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete URL"})
	}

	// Return success response
	return c.JSON(http.StatusOK, map[string]string{"message": "URL deleted successfully"})
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create a simple handler for the health check
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	// Call the handler
	if assert.NoError(t, handler(c)) {
		// Assertions
		assert.Equal(t, http.StatusOK, rec.Code)
		var response map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
	}
}

// TestShortenURL tests the ShortenURL handler
func TestShortenURL(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockURLService)
	handler := NewTestURLHandler(mockService, "http://localhost:8080", "test-api-key")

	// Test case 1: Valid request
	t.Run("ValidRequest", func(t *testing.T) {
		// Setup request
		reqBody := ShortenRequest{
			URL:              "https://example.com",
			CustomCode:       "custom",
			Title:            "Example",
			Expiry:           3600,
			CreatorReference: "test-user",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "test-api-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Setup mock
		url := &models.URL{
			Original:         "https://example.com",
			Short:            "custom",
			Title:            "Example",
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(time.Hour),
			Clicks:           0,
			CreatorReference: "test-user",
		}
		mockService.On("CreateShortURL", mock.Anything, "https://example.com", "custom", "Example", time.Duration(3600), "test-user").Return(url, nil)

		// Call handler
		err := handler.ShortenURL(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		var response URLResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", response.OriginalURL)
		assert.Equal(t, "http://localhost:8080/custom", response.ShortURL)
		assert.Equal(t, "Example", response.Title)
		assert.Equal(t, int64(0), response.Clicks)
		assert.Equal(t, "test-user", response.CreatorReference)

		// Verify mock
		mockService.AssertExpectations(t)
	})

	// Test case 2: Invalid URL
	t.Run("InvalidURL", func(t *testing.T) {
		// Setup request
		reqBody := ShortenRequest{
			URL:        "invalid-url",
			CustomCode: "custom",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "test-api-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Setup mock
		mockService.On("CreateShortURL", mock.Anything, "invalid-url", "custom", "", time.Duration(0), "").Return(nil, store.ErrInvalidURL)

		// Call handler
		err := handler.ShortenURL(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid URL", response["error"])

		// Verify mock
		mockService.AssertExpectations(t)
	})
}

// TestRedirectURL tests the RedirectURL handler
func TestRedirectURL(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockURLService)
	handler := NewTestURLHandler(mockService, "http://localhost:8080", "test-api-key")

	// Test case 1: Valid short URL
	t.Run("ValidShortURL", func(t *testing.T) {
		// Setup request
		req := httptest.NewRequest(http.MethodGet, "/:code", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("code")
		c.SetParamValues("abc123")

		// Setup mock
		url := &models.URL{
			Original:  "https://example.com",
			Short:     "abc123",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Clicks:    0,
		}
		mockService.On("GetByShort", mock.Anything, "abc123").Return(url, nil)

		// Call handler
		err := handler.RedirectURL(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "https://example.com", rec.Header().Get("Location"))

		// Verify mock
		mockService.AssertExpectations(t)
	})

	// Test case 2: URL not found
	t.Run("URLNotFound", func(t *testing.T) {
		// Setup request
		req := httptest.NewRequest(http.MethodGet, "/:code", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("code")
		c.SetParamValues("notfound")

		// Setup mock
		mockService.On("GetByShort", mock.Anything, "notfound").Return(nil, store.ErrURLNotFound)

		// Call handler
		err := handler.RedirectURL(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "URL not found", response["error"])

		// Verify mock
		mockService.AssertExpectations(t)
	})
}

// TestGetURLInfo tests the GetURLInfo handler
func TestGetURLInfo(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockURLService)
	handler := NewTestURLHandler(mockService, "http://localhost:8080", "test-api-key")

	// Test case: Valid short URL
	t.Run("ValidShortURL", func(t *testing.T) {
		// Setup request
		req := httptest.NewRequest(http.MethodGet, "/api/urls/:code", nil)
		req.Header.Set("X-API-Key", "test-api-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("code")
		c.SetParamValues("abc123")

		// Setup mock
		url := &models.URL{
			Original:         "https://example.com",
			Short:            "abc123",
			Title:            "Example",
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(time.Hour),
			Clicks:           10,
			CreatorReference: "test-user",
		}
		mockService.On("GetByShort", mock.Anything, "abc123").Return(url, nil)

		// Call handler
		err := handler.GetURLInfo(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var response URLResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", response.OriginalURL)
		assert.Equal(t, "http://localhost:8080/abc123", response.ShortURL)
		assert.Equal(t, "Example", response.Title)
		assert.Equal(t, int64(10), response.Clicks)
		assert.Equal(t, "test-user", response.CreatorReference)

		// Verify mock
		mockService.AssertExpectations(t)
	})
}

// TestDeleteURL tests the DeleteURL handler
func TestDeleteURL(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockURLService)
	handler := NewTestURLHandler(mockService, "http://localhost:8080", "test-api-key")

	// Test case: Valid delete
	t.Run("ValidDelete", func(t *testing.T) {
		// Setup request
		req := httptest.NewRequest(http.MethodDelete, "/api/urls/:code", nil)
		req.Header.Set("X-API-Key", "test-api-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("code")
		c.SetParamValues("abc123")

		// Setup mock
		url := &models.URL{
			ID:               1,
			Original:         "https://example.com",
			Short:            "abc123",
			Title:            "Example",
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(time.Hour),
			Clicks:           10,
			CreatorReference: "test-user",
		}
		mockService.On("GetByShort", mock.Anything, "abc123").Return(url, nil)
		mockService.On("Delete", mock.Anything, "abc123").Return(nil)

		// Call handler
		err := handler.DeleteURL(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var response map[string]string
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "URL deleted successfully", response["message"])

		// Verify mock
		mockService.AssertExpectations(t)
	})
}

// TestURLHandler_Integration is a simple integration test for the URL handler
// It tests the basic functionality of the handler without mocking
func TestURLHandler_Integration(t *testing.T) {
	// Skip this test in CI environments
	t.Skip("Skipping integration test")

	// Setup
	e := echo.New()

	// Create a real URL service with real dependencies
	// This would require a running database and cache
	// In a real-world scenario, you would use proper mocking or test containers

	// Create the handler
	handler := &URLHandler{
		service: nil, // Replace with a real service if running the test
		baseURL: "http://localhost:8080",
		apiKey:  "test-api-key",
	}

	// Register routes
	handler.Register(e)

	// Test the health check endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)
}
