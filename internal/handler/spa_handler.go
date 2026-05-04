package handler

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

func RegisterSPA(e *echo.Echo, distFS fs.FS) {
	staticFS, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		return
	}

	assetHandler := http.FileServer(http.FS(staticFS))

	e.GET("/assets/*", echo.WrapHandler(http.StripPrefix("/assets", assetHandler)))

	e.GET("/vite-manifest.json", echo.WrapHandler(assetHandler))
	e.GET("/vite.svg", echo.WrapHandler(assetHandler))

	e.GET("/admin/*", func(c echo.Context) error {
		path := c.Param("*")
		if path != "" && !strings.HasSuffix(path, "/") {
			if f, err := staticFS.Open(path); err == nil {
				f.Close()
				c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
				http.StripPrefix("/admin", assetHandler).ServeHTTP(c.Response().Writer, c.Request())
				return nil
			}
		}

		indexHTML, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			return c.String(http.StatusNotFound, "index.html not found")
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		c.Response().WriteHeader(http.StatusOK)
		c.Response().Write(indexHTML)
		return nil
	})
}

func IsDistDirPresent() bool {
	_, err := os.Stat(filepath.Join("web", "dist"))
	return err == nil
}