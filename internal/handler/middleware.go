package handler

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

func APIKeyMiddleware(apiKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-API-Key")
			if key == "" || key != apiKey {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or missing API key",
				})
			}
			return next(c)
		}
	}
}

func SessionOrAPIKeyMiddleware(store *sessions.CookieStore, apiKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-API-Key")
			if key != "" {
				if key == apiKey {
					return next(c)
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or missing API key",
				})
			}

			session, err := store.Get(c.Request(), sessionName)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid session",
				})
			}

			authenticated, _ := session.Values["authenticated"].(bool)
			if !authenticated {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authentication required",
				})
			}

			return next(c)
		}
	}
}