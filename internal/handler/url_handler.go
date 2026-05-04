package handler

import (
	"context"
	_ "embed"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/fransfilastap/urlshortener/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

//go:embed redirect.html
var redirectHTML string

//go:embed error.html
var errorHTML string

//go:embed index.html
var indexHTML string

var redirectTmpl = template.Must(template.New("redirect").Parse(redirectHTML))
var errorTmpl = template.Must(template.New("error").Parse(errorHTML))
var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

type URLServicer interface {
	CreateShortURL(ctx context.Context, originalURL string, customShort string, title string, expireAfter time.Duration, creatorReference string) (*domain.URL, error)
	GetByShort(ctx context.Context, short string) (*domain.URL, error)
	GetByOriginal(ctx context.Context, original string) (*domain.URL, error)
	GetByCreator(ctx context.Context, creatorReference string) ([]*domain.URL, error)
	IncrementClicks(ctx context.Context, short string) error
	Delete(ctx context.Context, short string) error
	DeleteWithCreator(ctx context.Context, short string, creatorReference string) error
	RecordClick(ctx context.Context, short string, ip, location, browser, device string) error
	GetClicksByShort(ctx context.Context, short string) ([]*domain.Click, error)
	GetClickAnalytics(ctx context.Context, short string) (map[string]interface{}, error)
	UpdateURL(ctx context.Context, short string, title, originalURL string, expireAfter time.Duration) (*domain.URL, error)
	UpdateURLWithCreator(ctx context.Context, short string, title, originalURL string, expireAfter time.Duration, creatorReference string) (*domain.URL, error)
}

type ShortenRequest struct {
	URL              string        `json:"url" validate:"required,url"`
	CustomCode       string        `json:"custom_code,omitempty"`
	Title            string        `json:"title,omitempty"`
	Expiry           time.Duration `json:"expiry,omitempty"`
	CreatorReference string        `json:"creator_reference,omitempty"`
}

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

type UpdateURLRequest struct {
	URL              string        `json:"url,omitempty"`
	Title            string        `json:"title,omitempty"`
	Expiry           time.Duration `json:"expiry,omitempty"`
	CreatorReference string        `json:"creator_reference,omitempty"`
}

type URLHandler struct {
	service URLServicer
	baseURL string
	apiKey  string
}

func NewURLHandler(service URLServicer, baseURL string, apiKey string) *URLHandler {
	return &URLHandler{
		service: service,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

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

	expiry := req.Expiry * time.Second
	url, err := h.service.CreateShortURL(c.Request().Context(), req.URL, req.CustomCode, req.Title, expiry, req.CreatorReference)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidURL):
			log.Error().Err(err).Str("url", req.URL).Msg("Invalid URL provided")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid URL"})
		case errors.Is(err, domain.ErrURLExists):
			log.Error().Err(err).Str("custom_code", req.CustomCode).Msg("Custom code already in use")
			return c.JSON(http.StatusConflict, map[string]string{"error": "Custom code already in use"})
		default:
			log.Error().Err(err).Str("url", req.URL).Msg("Failed to create short URL")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create short URL"})
		}
	}

	shortURL := h.baseURL + "/" + url.Short

	log.Info().
		Str("original_url", url.Original).
		Str("short_url", shortURL).
		Time("expires_at", url.ExpiresAt).
		Msg("URL shortened successfully")

	return c.JSON(http.StatusCreated, URLResponse{
		OriginalURL:      url.Original,
		ShortURL:         shortURL,
		Title:            url.Title,
		ExpiresAt:        url.ExpiresAt,
		Clicks:           url.Clicks,
		CreatorReference: url.CreatorReference,
	})
}

func (h *URLHandler) RedirectURL(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in redirect request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	log.Debug().Str("code", code).Msg("Redirecting short URL")

	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for redirect")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for redirect")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	go func() {
		ctx := context.Background()
		req := c.Request()
		ip := c.RealIP()
		userAgent := req.UserAgent()

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

		location := "Unknown"

		err := h.service.RecordClick(ctx, code, ip, location, browser, device)
		if err != nil {
			if errors.Is(err, domain.ErrRecentClick) {
				log.Debug().Str("code", code).Msg("Recent click from the same visitor, not incrementing click count")
			} else {
				log.Error().Err(err).Str("code", code).Msg("Failed to record click analytics")
			}
		} else {
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

	if strings.Contains(c.Request().Header.Get("Accept"), "text/html") {
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

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		c.Response().WriteHeader(http.StatusOK)
		if err := redirectTmpl.Execute(c.Response().Writer, data); err != nil {
			log.Error().Err(err).Msg("Failed to render template")
			return c.Redirect(http.StatusFound, url.Original)
		}

		return nil
	}

	return c.Redirect(http.StatusFound, url.Original)
}

func (h *URLHandler) GetURLInfo(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in info request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	log.Debug().Str("code", code).Msg("Getting URL info")

	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for info request")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for info request")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	shortURL := h.baseURL + "/" + url.Short

	log.Info().
		Str("code", code).
		Str("original_url", url.Original).
		Str("short_url", shortURL).
		Time("expires_at", url.ExpiresAt).
		Int64("clicks", url.Clicks).
		Msg("URL info retrieved")

	return c.JSON(http.StatusOK, URLResponse{
		OriginalURL:      url.Original,
		ShortURL:         shortURL,
		Title:            url.Title,
		ExpiresAt:        url.ExpiresAt,
		Clicks:           url.Clicks,
		CreatorReference: url.CreatorReference,
	})
}

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

	existingURL, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for update")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for update")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	originalURL := existingURL.Original
	if req.URL != "" {
		originalURL = req.URL
	}

	title := existingURL.Title
	if req.Title != "" {
		title = req.Title
	}

	var updatedURL *domain.URL
	var updateErr error

	if req.CreatorReference != "" {
		expiry := req.Expiry * time.Second
		updatedURL, updateErr = h.service.UpdateURLWithCreator(c.Request().Context(), code, title, originalURL, expiry, req.CreatorReference)
	} else {
		log.Warn().Str("code", code).Msg("No creator reference provided for URL update")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing creator reference"})
	}

	if updateErr != nil {
		switch {
		case errors.Is(updateErr, domain.ErrInvalidURL):
			log.Error().Err(updateErr).Str("url", req.URL).Msg("Invalid URL provided")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid URL"})
		case errors.Is(updateErr, domain.ErrURLNotFound):
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

	shortURL := h.baseURL + "/" + updatedURL.Short

	log.Info().
		Str("code", code).
		Str("original_url", updatedURL.Original).
		Str("short_url", shortURL).
		Str("title", updatedURL.Title).
		Time("expires_at", updatedURL.ExpiresAt).
		Msg("URL updated successfully")

	return c.JSON(http.StatusOK, URLResponse{
		OriginalURL:      updatedURL.Original,
		ShortURL:         shortURL,
		Title:            updatedURL.Title,
		ExpiresAt:        updatedURL.ExpiresAt,
		Clicks:           updatedURL.Clicks,
		CreatorReference: updatedURL.CreatorReference,
	})
}

func (h *URLHandler) DeleteURL(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in delete request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	creatorReference := c.QueryParam("creator_reference")

	log.Debug().
		Str("code", code).
		Str("creator_reference", creatorReference).
		Msg("Deleting URL")

	var err error

	if creatorReference != "" {
		err = h.service.DeleteWithCreator(c.Request().Context(), code, creatorReference)
	} else {
		log.Warn().Str("code", code).Msg("No creator reference provided for URL deletion")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing creator reference"})
	}

	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
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

	return c.JSON(http.StatusOK, map[string]string{"message": "URL deleted successfully"})
}

func (h *URLHandler) GetURLAnalytics(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		log.Error().Msg("Missing URL code in analytics request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	log.Debug().Str("code", code).Msg("Getting URL analytics")

	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for analytics request")
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for analytics request")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}

	analytics, err := h.service.GetClickAnalytics(c.Request().Context(), code)
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve analytics data")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve analytics data"})
	}

	clicks, err := h.service.GetClicksByShort(c.Request().Context(), code)
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve click data")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve click data"})
	}

	maxClicks := 100
	if len(clicks) > maxClicks {
		clicks = clicks[:maxClicks]
	}

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

func (h *URLHandler) GetURLsByCreator(c echo.Context) error {
	creatorReference := c.Param("creator_reference")
	if creatorReference == "" {
		log.Error().Msg("Missing creator reference in request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing creator reference"})
	}

	log.Debug().Str("creator_reference", creatorReference).Msg("Getting URLs by creator")

	urls, err := h.service.GetByCreator(c.Request().Context(), creatorReference)
	if err != nil {
		log.Error().Err(err).Str("creator_reference", creatorReference).Msg("Failed to retrieve URLs by creator")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URLs by creator"})
	}

	if len(urls) == 0 {
		log.Info().Str("creator_reference", creatorReference).Msg("No URLs found for creator")
		return c.JSON(http.StatusOK, []interface{}{})
	}

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

	return c.JSON(http.StatusOK, response)
}