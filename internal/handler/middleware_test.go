package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeyMiddleware(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	apiKey := "test-api-key"

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	middleware := APIKeyMiddleware(apiKey)
	middlewareHandler := middleware(handler)

	t.Run("NoAPIKey", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middlewareHandler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid or missing API key")
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "invalid-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middlewareHandler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid or missing API key")
	})

	t.Run("ValidAPIKey", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", apiKey)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middlewareHandler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "success", rec.Body.String())
	})
}