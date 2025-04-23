package handlers

import (
	"context"
	"errors"
	"github.com/fransfilastap/urlshortener/models"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/fransfilastap/urlshortener/store"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// ShortenRequest represents a request to shorten a URL
type ShortenRequest struct {
	URL              string        `json:"url" validate:"required,url"`
	CustomCode       string        `json:"custom_code,omitempty"`
	Title            string        `json:"title,omitempty"`
	Expiry           time.Duration `json:"expiry,omitempty"` // in seconds
	CreatorReference string        `json:"creator_reference,omitempty"`
}

// URLResponse represents a response with URL information
type URLResponse struct {
	OriginalURL      string    `json:"original_url"`
	ShortURL         string    `json:"short_url"`
	ShortCode        string    `json:"short_code"`
	Title            string    `json:"title,omitempty"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	Clicks           int64     `json:"clicks"`
	CreatorReference string    `json:"creator_reference,omitempty"`
}

// URLHandler handles URL shortening requests
type URLHandler struct {
	service *store.URLService
	baseURL string
	apiKey  string
}

// NewURLHandler creates a new URL handler
func NewURLHandler(service *store.URLService, baseURL string, apiKey string) *URLHandler {
	return &URLHandler{
		service: service,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// Register registers the URL handler routes with Echo
func (h *URLHandler) Register(e *echo.Echo) {
	// Public endpoint for redirecting
	e.GET("/:code", h.RedirectURL)

	// Protected endpoints that require API key
	apiGroup := e.Group("")
	apiGroup.Use(APIKeyMiddleware(h.apiKey))
	apiGroup.POST("/api/shorten", h.ShortenURL)
	apiGroup.GET("/api/urls/:code", h.GetURLInfo)
	apiGroup.PUT("/api/urls/:code", h.UpdateURL)
	apiGroup.DELETE("/api/urls/:code", h.DeleteURL)
	apiGroup.GET("/api/urls/:code/analytics", h.GetURLAnalytics)
	apiGroup.GET("/api/urls/creator/:creator_reference", h.GetURLsByCreator)
}

// ShortenURL handles requests to create short URLs
func (h *URLHandler) ShortenURL(c echo.Context) error {
	var req ShortenRequest
	if err := c.Bind(&req); err != nil {
		log.Error().Err(err).Msg("Invalid request format for URL shortening")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	log.Debug().
		Str("original_url", req.URL).
		Str("custom_code", req.CustomCode).
		Str("title", req.Title).
		Dur("expiry", req.Expiry).
		Str("creator_reference", req.CreatorReference).
		Msg("Shortening URL")

	// Create short URL
	// Convert expiry from seconds to time.Duration
	expiry := req.Expiry * time.Second
	url, err := h.service.CreateShortURL(c.Request().Context(), req.URL, req.CustomCode, req.Title, expiry, req.CreatorReference)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrInvalidURL):
			log.Error().Err(err).Str("url", req.URL).Msg("Invalid URL provided")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid URL"})
		case errors.Is(err, store.ErrURLExists):
			log.Error().Err(err).Str("custom_code", req.CustomCode).Msg("Custom code already in use")
			return c.JSON(http.StatusConflict, map[string]string{"error": "Custom code already in use"})
		default:
			log.Error().Err(err).Str("url", req.URL).Msg("Failed to create short URL")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create short URL"})
		}
	}

	// Construct full short URL
	shortURL := h.baseURL + "/" + url.Short

	log.Info().
		Str("original_url", url.Original).
		Str("short_url", shortURL).
		Time("expires_at", url.ExpiresAt).
		Msg("URL shortened successfully")

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
func (h *URLHandler) RedirectURL(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in redirect request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	log.Debug().Str("code", code).Msg("Redirecting short URL")

	// Get URL by short code
	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, store.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for redirect")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for redirect")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	// Increment click count and record analytics asynchronously
	go func() {
		ctx := context.Background()

		// Extract client information
		req := c.Request()
		ip := c.RealIP()
		userAgent := req.UserAgent()

		// Simple parsing of user agent - in a real app, you'd use a proper user agent parser library
		var browser, device string
		if strings.Contains(userAgent, "Mozilla") {
			browser = "Mozilla"
		} else if strings.Contains(userAgent, "Chrome") {
			browser = "Chrome"
		} else if strings.Contains(userAgent, "Safari") {
			browser = "Safari"
		} else if strings.Contains(userAgent, "Edge") {
			browser = "Edge"
		} else if strings.Contains(userAgent, "Firefox") {
			browser = "Firefox"
		} else {
			browser = "Other"
		}

		if strings.Contains(userAgent, "Mobile") {
			device = "Mobile"
		} else if strings.Contains(userAgent, "Tablet") {
			device = "Tablet"
		} else {
			device = "Desktop"
		}

		// Simple location determination based on IP - in a real app, you'd use a geolocation service
		location := "Unknown"

		// Record click analytics
		err := h.service.RecordClick(ctx, code, ip, location, browser, device)
		if err != nil {
			if errors.Is(err, store.ErrRecentClick) {
				log.Debug().Str("code", code).Msg("Recent click from the same visitor, not incrementing click count")
			} else {
				log.Error().Err(err).Str("code", code).Msg("Failed to record click analytics")
			}
		} else {
			// Only increment click count if it's a unique click or if the last click from the same visitor was more than 1 hour ago
			if err := h.service.IncrementClicks(ctx, code); err != nil {
				log.Error().Err(err).Str("code", code).Msg("Failed to increment click count")
			}
		}
	}()

	log.Info().
		Str("code", code).
		Str("original_url", url.Original).
		Int64("clicks", url.Clicks+1).
		Msg("Serving redirect page for URL")

	// Check if the request accepts HTML
	if strings.Contains(c.Request().Header.Get("Accept"), "text/html") {
		// Define template data
		type TemplateData struct {
			OriginalURL string
			ShortURL    string
			Clicks      int64
		}

		data := TemplateData{
			OriginalURL: url.Original,
			ShortURL:    url.Short,
			Clicks:      url.Clicks,
		}

		// Parse the template
		tmpl, err := template.ParseFiles("static/redirect.html")
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse template")
			return c.Redirect(http.StatusFound, url.Original)
		}

		// Render the template
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		c.Response().WriteHeader(http.StatusOK)
		if err := tmpl.Execute(c.Response().Writer, data); err != nil {
			log.Error().Err(err).Msg("Failed to render template")
			return c.Redirect(http.StatusFound, url.Original)
		}

		return nil
	}

	// For non-HTML requests (API clients, etc.), perform a direct redirect
	return c.Redirect(http.StatusFound, url.Original)
}

// GetURLInfo returns information about a short URL
func (h *URLHandler) GetURLInfo(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in info request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	log.Debug().Str("code", code).Msg("Getting URL info")

	// Get URL by short code
	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, store.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for info request")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for info request")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	// Construct full short URL
	shortURL := h.baseURL + "/" + url.Short

	log.Info().
		Str("code", code).
		Str("original_url", url.Original).
		Str("short_url", shortURL).
		Time("expires_at", url.ExpiresAt).
		Int64("clicks", url.Clicks).
		Msg("URL info retrieved")

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

// UpdateURLRequest represents a request to update a URL
type UpdateURLRequest struct {
	URL              string        `json:"url,omitempty"`
	Title            string        `json:"title,omitempty"`
	Expiry           time.Duration `json:"expiry,omitempty"` // in seconds
	CreatorReference string        `json:"creator_reference,omitempty"`
}

// UpdateURL handles requests to update a URL
func (h *URLHandler) UpdateURL(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in update request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	var req UpdateURLRequest
	if err := c.Bind(&req); err != nil {
		log.Error().Err(err).Msg("Invalid request format for URL update")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	log.Debug().
		Str("code", code).
		Str("url", req.URL).
		Str("title", req.Title).
		Dur("expiry", req.Expiry).
		Str("creator_reference", req.CreatorReference).
		Msg("Updating URL")

	// Get existing URL to verify it exists
	existingURL, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, store.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for update")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for update")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	// Use existing values if not provided in request
	originalURL := existingURL.Original
	if req.URL != "" {
		originalURL = req.URL
	}

	title := existingURL.Title
	if req.Title != "" {
		title = req.Title
	}

	var updatedURL *models.URL
	var updateErr error

	// If creator reference is provided, use it for authorization
	if req.CreatorReference != "" {
		// Convert expiry from seconds to time.Duration
		expiry := req.Expiry * time.Second
		// Update URL with creator reference check
		updatedURL, updateErr = h.service.UpdateURLWithCreator(c.Request().Context(), code, title, originalURL, expiry, req.CreatorReference)
	} else {
		log.Warn().Str("code", code).Msg("No creator reference provided for URL update")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing creator reference"})
	}

	if updateErr != nil {
		switch {
		case errors.Is(updateErr, store.ErrInvalidURL):
			log.Error().Err(updateErr).Str("url", req.URL).Msg("Invalid URL provided")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid URL"})
		case errors.Is(updateErr, store.ErrURLNotFound):
			log.Error().Err(updateErr).Str("code", code).Msg("URL not found for update")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		case strings.Contains(updateErr.Error(), "unauthorized"):
			log.Error().Err(updateErr).Str("code", code).Str("creator_reference", req.CreatorReference).Msg("Unauthorized update attempt")
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized: creator reference does not match"})
		default:
			log.Error().Err(updateErr).Str("code", code).Msg("Failed to update URL")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update URL"})
		}
	}

	// Construct full short URL
	shortURL := h.baseURL + "/" + updatedURL.Short

	log.Info().
		Str("code", code).
		Str("original_url", updatedURL.Original).
		Str("short_url", shortURL).
		Str("title", updatedURL.Title).
		Time("expires_at", updatedURL.ExpiresAt).
		Msg("URL updated successfully")

	// Return response
	return c.JSON(http.StatusOK, URLResponse{
		OriginalURL:      updatedURL.Original,
		ShortURL:         shortURL,
		Title:            updatedURL.Title,
		ExpiresAt:        updatedURL.ExpiresAt,
		Clicks:           updatedURL.Clicks,
		CreatorReference: updatedURL.CreatorReference,
	})
}

// DeleteURL handles requests to delete a URL
func (h *URLHandler) DeleteURL(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in delete request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	// Check for creator reference in query parameters
	creatorReference := c.QueryParam("creator_reference")

	log.Debug().
		Str("code", code).
		Str("creator_reference", creatorReference).
		Msg("Deleting URL")

	var err error

	// If creator reference is provided, use it for authorization
	if creatorReference != "" {
		// Delete URL with creator reference check
		err = h.service.DeleteWithCreator(c.Request().Context(), code, creatorReference)
	} else {
		// Delete URL without creator reference check (legacy behavior)
		log.Warn().Str("code", code).Msg("No creator reference provided for URL deletion")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing creator reference"})
	}

	if err != nil {
		if errors.Is(err, store.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for deletion")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		} else if strings.Contains(err.Error(), "unauthorized") {
			log.Error().Err(err).Str("code", code).Str("creator_reference", creatorReference).Msg("Unauthorized delete attempt")
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized: creator reference does not match"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to delete URL")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete URL"})
	}

	log.Info().
		Str("code", code).
		Str("creator_reference", creatorReference).
		Msg("URL deleted successfully")

	// Return success response
	return c.JSON(http.StatusOK, map[string]string{"message": "URL deleted successfully"})
}

// GetURLAnalytics returns analytics data for a URL
func (h *URLHandler) GetURLAnalytics(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in analytics request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	log.Debug().Str("code", code).Msg("Getting URL analytics")

	// Get URL to verify it exists
	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, store.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for analytics request")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for analytics request")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	// Get analytics data
	analytics, err := h.service.GetClickAnalytics(c.Request().Context(), code)
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve analytics data")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve analytics data"})
	}

	// Get detailed click data (limited to last 100 for performance)
	clicks, err := h.service.GetClicksByShort(c.Request().Context(), code)
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve click data")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve click data"})
	}

	// Limit to last 100 clicks
	maxClicks := 100
	if len(clicks) > maxClicks {
		clicks = clicks[:maxClicks]
	}

	// Combine data
	result := map[string]interface{}{
		"url": URLResponse{
			OriginalURL:      url.Original,
			ShortURL:         h.baseURL + "/" + url.Short,
			Title:            url.Title,
			ExpiresAt:        url.ExpiresAt,
			Clicks:           url.Clicks,
			CreatorReference: url.CreatorReference,
		},
		"analytics":     analytics,
		"recent_clicks": clicks,
	}

	log.Info().
		Str("code", code).
		Int("total_clicks", len(clicks)).
		Msg("URL analytics retrieved")

	return c.JSON(http.StatusOK, result)
}

// GetURLsByCreator returns all URLs created by a specific creator
func (h *URLHandler) GetURLsByCreator(c echo.Context) error {
	creatorReference := c.Param("creator_reference")
	if creatorReference == "" {
		log.Error().Msg("Missing creator reference in request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing creator reference"})
	}

	log.Debug().Str("creator_reference", creatorReference).Msg("Getting URLs by creator")

	// Get URLs by creator
	urls, err := h.service.GetByCreator(c.Request().Context(), creatorReference)
	if err != nil {
		log.Error().Err(err).Str("creator_reference", creatorReference).Msg("Failed to retrieve URLs by creator")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URLs by creator"})
	}

	if len(urls) == 0 {
		log.Info().Str("creator_reference", creatorReference).Msg("No URLs found for creator")
		return c.JSON(http.StatusOK, []interface{}{})
	}

	// Convert URLs to response format
	var response []URLResponse
	for _, url := range urls {
		shortURL := h.baseURL + "/" + url.Short
		response = append(response, URLResponse{
			OriginalURL:      url.Original,
			ShortURL:         shortURL,
			ShortCode:        url.Short,
			Title:            url.Title,
			ExpiresAt:        url.ExpiresAt,
			CreatedAt:        url.CreatedAt,
			Clicks:           url.Clicks,
			CreatorReference: url.CreatorReference,
		})
	}

	log.Info().
		Str("creator_reference", creatorReference).
		Int("count", len(urls)).
		Msg("URLs retrieved by creator successfully")

	// Return URLs
	return c.JSON(http.StatusOK, response)
}
