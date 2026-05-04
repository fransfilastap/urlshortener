package handler

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

const sessionName = "urlshortener-session"

type AuthHandler struct {
	store  *sessions.CookieStore
	apiKey string
}

func NewAuthHandler(store *sessions.CookieStore, apiKey string) *AuthHandler {
	return &AuthHandler{
		store:  store,
		apiKey: apiKey,
	}
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req struct {
		APIKey string `json:"api_key"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.APIKey != h.apiKey {
		log.Warn().Msg("Failed login attempt")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid API key"})
	}

	session, _ := h.store.Get(c.Request(), sessionName)
	session.Values["authenticated"] = true
	session.Values["api_key"] = req.APIKey
	session.Options.MaxAge = 86400
	session.Options.HttpOnly = true
	session.Options.SameSite = http.SameSiteStrictMode
	session.Options.Path = "/"

	if err := session.Save(c.Request(), c.Response()); err != nil {
		log.Error().Err(err).Msg("Failed to save session")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create session"})
	}

	log.Info().Msg("User logged in successfully")
	return c.JSON(http.StatusOK, map[string]string{"message": "Logged in successfully"})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	session, _ := h.store.Get(c.Request(), sessionName)
	session.Options.MaxAge = -1
	session.Values["authenticated"] = false

	if err := session.Save(c.Request(), c.Response()); err != nil {
		log.Error().Err(err).Msg("Failed to clear session")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to logout"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func (h *AuthHandler) Me(c echo.Context) error {
	session, _ := h.store.Get(c.Request(), sessionName)
	authenticated, _ := session.Values["authenticated"].(bool)

	if !authenticated {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Not authenticated"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"authenticated": true,
	})
}

func GetSessionStore(secret string, maxAge int) *sessions.CookieStore {
	store := sessions.NewCookieStore([]byte(secret))
	store.Options.MaxAge = maxAge
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteStrictMode
	store.Options.Path = "/"
	return store
}