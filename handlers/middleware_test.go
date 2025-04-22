package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeyMiddleware(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	apiKey := "test-api-key"

	// Create a handler that will be called if the middleware passes
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	// Create the middleware
	middleware := APIKeyMiddleware(apiKey)
	middlewareHandler := middleware(handler)

	// Test case 1: No API key
	t.Run("NoAPIKey", func(t *testing.T) {
		// Reset recorder
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Call the middleware
		err := middlewareHandler(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid or missing API key")
	})

	// Test case 2: Invalid API key
	t.Run("InvalidAPIKey", func(t *testing.T) {
		// Reset recorder and set invalid API key
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "invalid-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Call the middleware
		err := middlewareHandler(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid or missing API key")
	})

	// Test case 3: Valid API key
	t.Run("ValidAPIKey", func(t *testing.T) {
		// Reset recorder and set valid API key
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", apiKey)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Call the middleware
		err := middlewareHandler(c)

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "success", rec.Body.String())
	})
}
