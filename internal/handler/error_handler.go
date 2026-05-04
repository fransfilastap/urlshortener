package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func CustomHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	message := "Internal Server Error"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if msg, ok := he.Message.(string); ok {
			message = msg
		}
	}

	acceptsHTML := strings.Contains(c.Request().Header.Get("Accept"), "text/html")

	if acceptsHTML {
		title := "Halaman Tidak Ditemukan"
		detail := "Alamat yang Anda tuju tidak tersedia."

		if code == http.StatusInternalServerError {
			title = "Terjadi Kesalahan"
			detail = "Server mengalami gangguan. Silakan coba beberapa saat lagi."
		} else if code == http.StatusBadRequest {
			title = "Permintaan Tidak Valid"
			detail = "Permintaan tidak dapat diproses."
		}

		data := ErrorTemplateData{
			ErrorCode:    code,
			ErrorTitle:   title,
			ErrorMessage: detail,
			ShortCode:    "",
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
		c.Response().WriteHeader(code)
		if renderErr := errorTmpl.Execute(c.Response().Writer, data); renderErr != nil {
			log.Error().Err(renderErr).Msg("Failed to render error page")
		}
		return
	}

	if err := c.JSON(code, map[string]string{"error": message}); err != nil {
		log.Error().Err(err).Msg("Failed to send error response")
	}
}