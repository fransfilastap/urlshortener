package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// APIKeyMiddleware creates a middleware that checks for a valid API key
func APIKeyMiddleware(apiKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get API key from header
			key := c.Request().Header.Get("X-API-Key")
			
			// Check if API key is valid
			if key == "" || key != apiKey {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or missing API key",
				})
			}
			
			// API key is valid, continue
			return next(c)
		}
	}
}