# Vite React Frontend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Vite + React + TailwindCSS admin SPA and fix the redirect page to use Go's `//go:embed` directive.

**Architecture:** The redirect page stays as a Go server-side template (embedded, not filesystem). The admin dashboard is a separate Vite React SPA that builds to `web/dist/` and gets embedded in the Go binary. Session-based auth uses `gorilla/sessions` for the web UI alongside the existing API key auth for programmatic access.

**Tech Stack:** Go (Echo), React 19, Vite 6, TailwindCSS 4, React Router v7, gorilla/sessions, Recharts

---

### Task 1: Fix redirect page embedding

**Files:**
- Move: `static/redirect.html` → `internal/handler/redirect.html`
- Modify: `internal/handler/url_handler.go`
- Remove: `cmd/urlshortener/main.go` (remove `e.Static("/static", "static")` line)

**Step 1: Move redirect.html to internal/handler/**

```bash
cp static/redirect.html internal/handler/redirect.html
```

**Step 2: Update redirect.html — add noscript fallback**

In `internal/handler/redirect.html`, add this line inside `<head>` before the closing `</head>`:

```html
<noscript>
    <meta http-equiv="refresh" content="0;url={{.OriginalURL}}">
</noscript>
```

**Step 3: Update url_handler.go — embed the template and parse from string**

In `internal/handler/url_handler.go`:

1. Add embed import — insert after the existing imports:
```go
import (
	"context"
	"embed"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"
	// ... existing imports
)
```

2. Add embed directive and parsed template at package level, after the imports:
```go
//go:embed redirect.html
var redirectHTML string

var redirectTmpl = template.Must(template.New("redirect").Parse(redirectHTML))
```

3. Replace the `RedirectURL` method's template section. Find this block:
```go
		tmpl, err := template.ParseFiles("static/redirect.html")
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse template")
			return c.Redirect(http.StatusFound, url.Original)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		c.Response().WriteHeader(http.StatusOK)
		if err := tmpl.Execute(c.Response().Writer, data); err != nil {
			log.Error().Err(err).Msg("Failed to render template")
			return c.Redirect(http.StatusFound, url.Original)
		}
```

Replace with:
```go
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		c.Response().WriteHeader(http.StatusOK)
		if err := redirectTmpl.Execute(c.Response().Writer, data); err != nil {
			log.Error().Err(err).Msg("Failed to render template")
			return c.Redirect(http.StatusFound, url.Original)
		}
```

**Step 4: Update cmd/urlshortener/main.go — remove static file serving**

Find and remove this line:
```go
	e.Static("/static", "static")
```

We'll re-add static serving from embedded FS in a later task.

**Step 5: Build and verify**

```bash
go build ./cmd/urlshortener/
```

Expected: clean build with no errors.

**Step 6: Run existing tests**

```bash
go test -short ./...
```

Expected: all tests pass (integration tests skipped).

**Step 7: Commit**

```bash
git add internal/handler/redirect.html internal/handler/url_handler.go cmd/urlshortener/main.go
git rm static/redirect.html
git commit -m "fix: embed redirect template in binary instead of filesystem read"
```

---

### Task 2: Add gorilla/sessions dependency and session auth

**Files:**
- Create: `internal/handler/auth_handler.go`
- Modify: `internal/handler/middleware.go`
- Modify: `internal/config/config.go`
- Modify: `go.mod` / `go.sum`

**Step 1: Install gorilla/sessions**

```bash
go get github.com/gorilla/sessions
go mod tidy
```

**Step 2: Add session config to config.go**

In `internal/config/config.go`, add these fields to the `Config` struct after `LogFormat`:

```go
	SessionSecret string
	SessionMaxAge  int
```

In `NewConfig()`, add after the `LogFormat` line:

```go
		SessionSecret: getEnv("SESSION_SECRET", "change-me-in-production"),
		SessionMaxAge: getEnvAsInt("SESSION_MAX_AGE", 86400),
```

**Step 3: Create auth_handler.go**

Create `internal/handler/auth_handler.go`:

```go
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
```

**Step 4: Add dual auth middleware to middleware.go**

In `internal/handler/middleware.go`, add a new function after `APIKeyMiddleware`:

```go
func SessionOrAPIKeyMiddleware(store *sessions.CookieStore, apiKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Try API key header first
			key := c.Request().Header.Get("X-API-Key")
			if key != "" {
				if key == apiKey {
					return next(c)
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or missing API key"})
			}

			// Try session cookie
			session, err := store.Get(c.Request(), "urlshortener-session")
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid session"})
			}

		 authenticated, _ := session.Values["authenticated"].(bool)
			if !authenticated {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
			}

			return next(c)
		}
	}
}
```

Add `"github.com/gorilla/sessions"` to the imports in middleware.go.

**Step 5: Build and verify**

```bash
go build ./cmd/urlshortener/
```

Expected: clean build.

**Step 6: Commit**

```bash
git add internal/handler/auth_handler.go internal/handler/middleware.go internal/config/config.go go.mod go.sum
git commit -m "feat: add session-based auth with gorilla/sessions"
```

---

### Task 3: Wire auth and sessions into main.go

**Files:**
- Modify: `cmd/urlshortener/main.go`

**Step 1: Update main.go to create session store and auth handler**

Replace the entire `cmd/urlshortener/main.go` with:

```go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fransfilastap/urlshortener"
	"github.com/fransfilastap/urlshortener/internal/config"
	"github.com/fransfilastap/urlshortener/internal/handler"
	"github.com/fransfilastap/urlshortener/internal/logger"
	"github.com/fransfilastap/urlshortener/internal/repository"
	"github.com/fransfilastap/urlshortener/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.NewConfig()

	logger.InitLogger(cfg.LogLevel, cfg.LogFormat)

	db, err := repository.NewPostgresRepository(cfg.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	if err := repository.RunMigrations(cfg.PostgresURL, urlshortener.MigrationsFS); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	cache := repository.NewCacheRepository(
		cfg.ValkeyCacheAddr,
		cfg.ValkeyCachePassword,
		cfg.ValkeyCacheDB,
		cfg.ValkeyCacheTTL,
	)
	defer cache.Close()

	urlService := service.NewURLService(db, cache)
	sessionStore := handler.GetSessionStore(cfg.SessionSecret, cfg.SessionMaxAge)
	authHandler := handler.NewAuthHandler(sessionStore, cfg.APIKey)

	e := echo.New()

	e.Use(logger.EchoLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Auth endpoints
	e.POST("/auth/login", authHandler.Login)
	e.POST("/auth/logout", authHandler.Logout)
	e.GET("/auth/me", authHandler.Me)

	// Public redirect endpoint
	urlHandler := handler.NewURLHandler(urlService, cfg.BaseURL, cfg.APIKey)
	e.GET("/:code", urlHandler.RedirectURL)

	// API endpoints — dual auth (session cookie OR API key header)
	apiGroup := e.Group("")
	apiGroup.Use(handler.SessionOrAPIKeyMiddleware(sessionStore, cfg.APIKey))
	apiGroup.POST("/api/shorten", urlHandler.ShortenURL)
	apiGroup.GET("/api/urls/:code", urlHandler.GetURLInfo)
	apiGroup.PUT("/api/urls/:code", urlHandler.UpdateURL)
	apiGroup.DELETE("/api/urls/:code", urlHandler.DeleteURL)
	apiGroup.GET("/api/urls/:code/analytics", urlHandler.GetURLAnalytics)
	apiGroup.GET("/api/urls/creator/:creator_reference", urlHandler.GetURLsByCreator)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// SPA and static assets will be mounted in a later task

	go func() {
		if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server shutdown failed")
	}

	log.Info().Msg("Server gracefully stopped")
}
```

Note: The `Register` method on URLHandler was removed in favor of explicit route registration in main.go. This means `url_handler_test.go`'s `TestURLHandler_Integration` test that calls `handler.Register(e)` needs to be updated later.

**Step 2: Remove Register method from url_handler.go**

In `internal/handler/url_handler.go`, remove the `Register` method:

```go
// Remove this entire method:
func (h *URLHandler) Register(e *echo.Echo) { ... }
```

**Step 3: Build and verify**

```bash
go build ./cmd/urlshortener/
```

**Step 4: Fix the handler test — remove Register reference**

In `internal/handler/url_handler_test.go`, find `TestURLHandler_Integration` and remove the line `handler.Register(e)` since routes are now registered in main.go.

**Step 5: Run tests**

```bash
go test -short ./...
```

**Step 6: Commit**

```bash
git add cmd/urlshortener/main.go internal/handler/url_handler.go internal/handler/url_handler_test.go
git commit -m "feat: wire session auth into main.go, remove Register method"
```

---

### Task 4: Create SPA handler for serving embedded frontend

**Files:**
- Create: `internal/handler/spa_handler.go`
- Modify: `embed.go`
- Modify: `cmd/urlshortener/main.go`

**Step 1: Update embed.go to add DistFS and StaticFS**

Replace `embed.go` with:

```go
package urlshortener

import "embed"

//go:embed db/migrations
var MigrationsFS embed.FS

//go:embed all:web/dist
var DistFS embed.FS

//go:embed static
var StaticFS embed.FS
```

Note: `all:web/dist` includes hidden files. The `web/dist` directory won't exist until the frontend is built, so we need to create a placeholder first:

```bash
mkdir -p web/dist
touch web/dist/.gitkeep
```

**Step 2: Create spa_handler.go**

Create `internal/handler/spa_handler.go`:

```go
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

	// Vite manifest and other build artifacts
	e.GET("/vite-manifest.json", echo.WrapHandler(assetHandler))
	e.GET("/vite.svg", echo.WrapHandler(assetHandler))

	// SPA fallback: any /admin/* route serves index.html
	e.GET("/admin/*", func(c echo.Context) error {
		// Try to serve a static file first
		path := c.Param("*")
		if path != "" && !strings.HasSuffix(path, "/") {
			// Check if the file exists in the embedded FS
			if f, err := staticFS.Open(path); err == nil {
				f.Close()
				c.Response().Header().Set("Cache-Control", "public, max-age=31536000")
				http.StripPrefix("/admin", assetHandler).ServeHTTP(c.Response().Writer, c.Request())
				return nil
			}
		}

		// Serve index.html for SPA routing
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

// IsDistDirPresent checks if web/dist exists (for development mode)
func IsDistDirPresent() bool {
	_, err := os.Stat(filepath.Join("web", "dist"))
	return err == nil
}
```

**Step 3: Update main.go to mount SPA and static**

In `cmd/urlshortener/main.go`, add the SPA handler after the health check route:

After `e.GET("/health", ...)` and before the SPA comment, add:

```go
	// Serve SPA (admin dashboard)
	if handler.IsDistDirPresent() {
		handler.RegisterSPA(e, urlshortener.DistFS)
	}

	// Serve static assets (logo etc)
	staticFS, _ := fs.Sub(urlshortener.StaticFS, "static")
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static", http.FileServer(http.FS(staticFS)))))
```

Add these imports to main.go:
```go
	"io/fs"
	"net/http"
```

**Step 4: Build and verify**

```bash
mkdir -p web/dist
touch web/dist/.gitkeep
go build ./cmd/urlshortener/
```

Expected: clean build.

**Step 5: Commit**

```bash
git add embed.go internal/handler/spa_handler.go cmd/urlshortener/main.go web/dist/.gitkeep
git commit -m "feat: add SPA handler for serving embedded frontend"
```

---

### Task 5: Scaffold Vite + React + TailwindCSS project

**Files:**
- Create: `web/` directory with full Vite project

**Step 1: Initialize Vite project**

```bash
cd web
npm create vite@latest . -- --template react-ts
npm install
```

**Step 2: Install dependencies**

```bash
npm install react-router-dom@7
npm install -D tailwindcss @tailwindcss/vite
```

**Step 3: Configure Tailwind**

Replace `web/vite.config.ts`:

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/auth': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    },
  },
})
```

**Step 4: Configure CSS**

Replace `web/src/index.css` with:

```css
@import "tailwindcss";
```

**Step 5: Create basic app structure**

Replace `web/src/main.tsx`:

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
)
```

Replace `web/src/App.tsx`:

```tsx
import { Routes, Route, Navigate } from 'react-router-dom'
import { lazy, Suspense } from 'react'

const Login = lazy(() => import('./pages/Login'))
const Dashboard = lazy(() => import('./pages/Dashboard'))

function App() {
  return (
    <Suspense fallback={<div className="flex items-center justify-center min-h-screen"><div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full"></div></div>}>
      <Routes>
        <Route path="/admin/login" element={<Login />} />
        <Route path="/admin/dashboard" element={<Dashboard />} />
        <Route path="/admin" element={<Navigate to="/admin/dashboard" replace />} />
        <Route path="*" element={<Navigate to="/admin/login" replace />} />
      </Routes>
    </Suspense>
  )
}

export default App
```

Create `web/src/pages/Login.tsx`:

```tsx
import { useState } from 'react'

export default function Login() {
  const [apiKey, setApiKey] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await fetch('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ api_key: apiKey }),
      })
      if (!res.ok) {
        const data = await res.json()
        setError(data.error || 'Login failed')
        return
      }
      window.location.href = '/admin/dashboard'
    } catch {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
        <h1 className="text-xl font-bold text-center mb-6">URL Shortener Admin</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="api-key" className="block text-sm font-medium text-gray-700">API Key</label>
            <input
              id="api-key"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              placeholder="Enter your API key"
            />
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 disabled:opacity-50 text-sm font-medium"
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  )
}
```

Create `web/src/pages/Dashboard.tsx`:

```tsx
export default function Dashboard() {
  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold">URL Shortener Dashboard</h1>
          <form action="/auth/logout" method="POST">
            <button type="submit" className="text-sm text-gray-500 hover:text-gray-700">Sign Out</button>
          </form>
        </div>
      </nav>
      <main className="max-w-7xl mx-auto px-6 py-8">
        <h2 className="text-2xl font-bold mb-6">Welcome</h2>
        <p className="text-gray-600">Dashboard content coming soon.</p>
      </main>
    </div>
  )
}
```

**Step 6: Create auth hook**

Create `web/src/hooks/useAuth.ts`:

```typescript
import { useState, useEffect } from 'react'

interface AuthState {
  authenticated: boolean
  loading: boolean
}

export function useAuth(): AuthState {
  const [auth, setAuth] = useState({ authenticated: false, loading: true })

  useEffect(() => {
    fetch('/auth/me', { credentials: 'include' })
      .then((res) => {
        if (res.ok) {
          setAuth({ authenticated: true, loading: false })
        } else {
          setAuth({ authenticated: false, loading: false })
        }
      })
      .catch(() => setAuth({ authenticated: false, loading: false }))
  }, [])

  return auth
}
```

**Step 7: Create API client**

Create `web/src/api/client.ts`:

```typescript
const BASE_URL = ''

export async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (res.status === 401) {
    window.location.href = '/admin/login'
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const data = await res.json().catch(() => ({ error: 'Request failed' }))
    throw new Error(data.error || 'Request failed')
  }

  return res.json()
}

export const api = {
  get: <T>(path: string) => apiFetch<T>(path),

  post: <T>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: 'POST', body: JSON.stringify(body) }),

  put: <T>(path: string, body: unknown) =>
    apiFetch<T>(path, { method: 'PUT', body: JSON.stringify(body) }),

  delete: <T>(path: string) => apiFetch<T>(path, { method: 'DELETE' }),
}
```

**Step 8: Update index.html title**

In `web/index.html`, update the `<title>` to "URL Shortener Admin".

**Step 9: Delete unused Vite template files**

```bash
rm -f web/src/App.css web/public/vite.svg
```

**Step 10: Build and verify frontend**

```bash
cd web && npm run build
ls dist/
```

Expected: `dist/` contains `index.html`, `assets/` directory.

**Step 11: Commit**

```bash
cd /Users/finnarc/Repo/urlshortener
git add web/
git commit -m "feat: scaffold Vite + React + TailwindCSS admin SPA"
```

---

### Task 6: Update Makefile and Dockerfile

**Files:**
- Modify: `Makefile`
- Modify: `Dockerfile`
- Modify: `.gitignore`

**Step 1: Add frontend targets to Makefile**

Add after the `build` target:

```makefile
# Install frontend dependencies
.PHONY: web-deps
web-deps:
	@echo "Installing frontend dependencies..."
	cd web && npm install

# Build frontend
.PHONY: web-build
web-build:
	@echo "Building frontend..."
	cd web && npm run build

# Dev frontend
.PHONY: web-dev
web-dev:
	@echo "Starting frontend dev server..."
	cd web && npm run dev

# Build everything (frontend + backend)
.PHONY: build-all
build-all: web-build build
```

Update the `clean` target to also clean frontend artifacts:

Find:
```makefile
clean:
	@echo "Cleaning..."
	$(GO) clean
	rm -f $(BINARY_NAME) $(BINARY_NAME)_linux_$(GOARCH) $(BINARY_NAME)_darwin_$(GOARCH) $(BINARY_NAME)_windows_$(GOARCH).exe $(COVERAGE_FILE)
```

Replace with:
```makefile
clean:
	@echo "Cleaning..."
	$(GO) clean
	rm -f $(BINARY_NAME) $(BINARY_NAME)_linux_$(GOARCH) $(BINARY_NAME)_darwin_$(GOARCH) $(BINARY_NAME)_windows_$(GOARCH).exe $(COVERAGE_FILE)
	rm -rf web/dist
```

**Step 2: Update Dockerfile for multi-stage frontend build**

Replace the entire `Dockerfile`:

```dockerfile
# Frontend build stage
FROM node:20-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Go build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o urlshortener ./cmd/urlshortener

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/urlshortener .
EXPOSE 8080
ENV SERVER_PORT="8080" \
    BASE_URL="http://localhost:8080" \
    API_KEY="your-api-key-here" \
    POSTGRES_URL="postgres://postgres:postgres@postgres:5432/urlshortener?sslmode=disable" \
    VALKEY_ADDR="valkey:6379" \
    VALKEY_PASSWORD="" \
    VALKEY_DB="0" \
    VALKEY_TTL="24h"
CMD ["./urlshortener"]
```

**Step 3: Update .gitignore**

Add to `.gitignore`:

```
# Frontend
web/node_modules
web/dist
web/.vite
```

**Step 4: Commit**

```bash
git add Makefile Dockerfile .gitignore
git commit -m "feat: update Makefile and Dockerfile for frontend build"
```

---

### Task 7: Build and test end-to-end

**Step 1: Build frontend**

```bash
cd web && npm install && npm run build && cd ..
```

**Step 2: Build Go binary**

```bash
go build ./cmd/urlshortener/
```

Expected: clean build.

**Step 3: Run all short tests**

```bash
go test -short ./...
```

Expected: all tests pass.

**Step 4: Start the server and verify**

```bash
export $(cat .env.example | grep -v '^#' | xargs)
./urlshortener
```

Test these endpoints:
- `GET /:code` — should render redirect template
- `GET /health` — should return `{"status":"ok"}`
- `GET /admin/login` — should serve React SPA (after frontend build)

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: verify full build pipeline works end-to-end"
```

---

### Task 8: Add URL management pages to the SPA

**Files:**
- Modify: `web/src/App.tsx`
- Create: `web/src/pages/UrlList.tsx`
- Create: `web/src/pages/UrlCreate.tsx`
- Create: `web/src/pages/UrlDetail.tsx`
- Create: `web/src/components/Layout.tsx`
- Create: `web/src/components/ProtectedRoute.tsx`

**Step 1: Create Layout component**

Create `web/src/components/Layout.tsx`:

```tsx
import { Link, Outlet } from 'react-router-dom'

export default function Layout() {
  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between max-w-7xl mx-auto">
          <div className="flex items-center gap-6">
            <Link to="/admin/dashboard" className="text-lg font-bold text-gray-900">URL Shortener</Link>
            <Link to="/admin/urls" className="text-sm text-gray-600 hover:text-gray-900">URLs</Link>
            <Link to="/admin/urls/create" className="text-sm text-gray-600 hover:text-gray-900">Create</Link>
          </div>
          <form action="/auth/logout" method="POST">
            <button type="submit" className="text-sm text-gray-500 hover:text-gray-700">Sign Out</button>
          </form>
        </div>
      </nav>
      <main className="max-w-7xl mx-auto px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
```

**Step 2: Create ProtectedRoute component**

Create `web/src/components/ProtectedRoute.tsx`:

```tsx
import { Navigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

export default function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { authenticated, loading } = useAuth()

  if (loading) {
    return <div className="flex items-center justify-center min-h-screen"><div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full"></div></div>
  }

  if (!authenticated) {
    return <Navigate to="/admin/login" replace />
  }

  return <>{children}</>
}
```

**Step 3: Create URL list page**

Create `web/src/pages/UrlList.tsx`:

```tsx
import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'

interface UrlItem {
  original_url: string
  short_url: string
  short_code: string
  title: string
  created_at: string
  clicks: number
  creator_reference: string
}

export default function UrlList() {
  const [urls, setUrls] = useState<UrlItem[]>([])
  const [loading, setLoading] = useState(true)
  const [creator, setCreator] = useState('')

  const fetchUrls = async (creatorRef?: string) => {
    setLoading(true)
    try {
      const path = creatorRef ? `/api/urls/creator/${creatorRef}` : '/api/urls/creator/all'
      const data = await api.get<UrlItem[]>(path)
      setUrls(Array.isArray(data) ? data : [])
    } catch {
      setUrls([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchUrls() }, [])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    fetchUrls(creator || undefined)
  }

  const handleDelete = async (code: string, creatorRef: string) => {
    if (!confirm('Delete this URL?')) return
    try {
      await api.delete(`/api/urls/${code}?creator_reference=${creatorRef}`)
      fetchUrls(creator || undefined)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Delete failed')
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">URLs</h2>
        <Link to="/admin/urls/create" className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700">Create URL</Link>
      </div>

      <form onSubmit={handleSearch} className="mb-6 flex gap-2">
        <input
          type="text"
          value={creator}
          onChange={(e) => setCreator(e.target.value)}
          placeholder="Filter by creator reference"
          className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
        />
        <button type="submit" className="bg-gray-100 px-4 py-2 rounded-md text-sm hover:bg-gray-200">Search</button>
      </form>

      {loading ? <div className="text-center py-8 text-gray-500">Loading...</div> : urls.length === 0 ? <div className="text-center py-8 text-gray-500">No URLs found.</div> : (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Short Code</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Original URL</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Clicks</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {urls.map((url) => (
                <tr key={url.short_code} className="hover:bg-gray-50">
                  <td className="px-6 py-4 text-sm"><Link to={`/admin/urls/${url.short_code}`} className="text-blue-600 hover:underline">{url.short_code}</Link></td>
                  <td className="px-6 py-4 text-sm text-gray-600 max-w-xs truncate">{url.original_url}</td>
                  <td className="px-6 py-4 text-sm text-gray-600">{url.clicks}</td>
                  <td className="px-6 py-4 text-sm"><button onClick={() => handleDelete(url.short_code, url.creator_reference)} className="text-red-600 hover:text-red-800">Delete</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
```

**Step 4: Create URL create page**

Create `web/src/pages/UrlCreate.tsx`:

```tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'

interface CreateResponse {
  original_url: string
  short_url: string
  short_code: string
  title: string
  clicks: number
}

export default function UrlCreate() {
  const navigate = useNavigate()
  const [url, setUrl] = useState('')
  const [customCode, setCustomCode] = useState('')
  const [title, setTitle] = useState('')
  const [creatorRef, setCreatorRef] = useState('')
  const [expiry, setExpiry] = useState(0)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<CreateResponse | null>(null)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    setResult(null)

    try {
      const body: Record<string, unknown> = { url }
      if (customCode) body.custom_code = customCode
      if (title) body.title = title
      if (creatorRef) body.creator_reference = creatorRef
      if (expiry > 0) body.expiry = expiry

      const data = await api.post<CreateResponse>('/api/shorten', body)
      setResult(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create URL')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Create Short URL</h2>

      {result && (
        <div className="mb-6 bg-green-50 border border-green-200 rounded-lg p-4">
          <p className="text-sm text-green-800">URL created successfully!</p>
          <p className="mt-2 text-lg font-mono"><a href={result.short_url} className="text-blue-600 hover:underline">{result.short_url}</a></p>
        </div>
      )}

      {error && <div className="mb-6 bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-800">{error}</div>}

      <form onSubmit={handleSubmit} className="bg-white rounded-lg border border-gray-200 p-6 space-y-4 max-w-lg">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Original URL *</label>
          <input type="url" required value={url} onChange={(e) => setUrl(e.target.value)} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" placeholder="https://example.com" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Custom Code</label>
          <input type="text" value={customCode} onChange={(e) => setCustomCode(e.target.value)} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" placeholder="my-custom-code" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input type="text" value={title} onChange={(e) => setTitle(e.target.value)} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Creator Reference</label>
          <input type="text" value={creatorRef} onChange={(e) => setCreatorRef(e.target.value)} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Expiry (seconds, 0 = never)</label>
          <input type="number" value={expiry} onChange={(e) => setExpiry(Number(e.target.value))} className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500" min={0} />
        </div>
        <div className="flex gap-3">
          <button type="submit" disabled={loading} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? 'Creating...' : 'Create'}</button>
          <button type="button" onClick={() => navigate(-1)} className="text-gray-600 px-4 py-2 rounded-md text-sm hover:bg-gray-100">Cancel</button>
        </div>
      </form>
    </div>
  )
}
```

**Step 5: Create URL detail page**

Create `web/src/pages/UrlDetail.tsx`:

```tsx
import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../api/client'

interface UrlInfo {
  original_url: string
  short_url: string
  short_code: string
  title: string
  expires_at: string
  created_at: string
  clicks: number
  creator_reference: string
}

interface Analytics {
  total_clicks: number
  browsers: Record<string, number>
  devices: Record<string, number>
  locations: Record<string, number>
}

export default function UrlDetail() {
  const { code } = useParams<{ code: string }>()
  const [url, setUrl] = useState<UrlInfo | null>(null)
  const [analytics, setAnalytics] = useState<Analytics | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!code) return
    api.get<UrlInfo>(`/api/urls/${code}`)
      .then(setUrl)
      .catch((err) => setError(err.message))

    api.get<Analytics>(`/api/urls/${code}/analytics`)
      .then(setAnalytics)
      .catch(() => {})
  }, [code])

  if (error) return <div className="text-center py-8 text-red-600">{error}</div>
  if (!url) return <div className="text-center py-8">Loading...</div>

  return (
    <div>
      <div className="flex items-center gap-2 mb-6">
        <Link to="/admin/urls" className="text-sm text-gray-500 hover:text-gray-700">&larr; Back to URLs</Link>
        <h2 className="text-2xl font-bold">URL Details</h2>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-6 space-y-4 max-w-2xl">
        <div><span className="text-sm text-gray-500">Short Code</span><p className="font-mono text-lg">{url.short_code}</p></div>
        <div><span className="text-sm text-gray-500">Original URL</span><p className="text-sm break-all"><a href={url.original_url} className="text-blue-600 hover:underline">{url.original_url}</a></p></div>
        <div><span className="text-sm text-gray-500">Short URL</span><p className="font-mono text-sm"><a href={url.short_url} className="text-blue-600 hover:underline">{url.short_url}</a></p></div>
        <div className="grid grid-cols-2 gap-4">
          <div><span className="text-sm text-gray-500">Title</span><p className="text-sm">{url.title || '—'}</p></div>
          <div><span className="text-sm text-gray-500">Total Clicks</span><p className="text-2xl font-bold">{url.clicks}</p></div>
          <div><span className="text-sm text-gray-500">Created</span><p className="text-sm">{new Date(url.created_at).toLocaleDateString()}</p></div>
          <div><span className="text-sm text-gray-500">Expires</span><p className="text-sm">{url.expires_at ? new Date(url.expires_at).toLocaleDateString() : 'Never'}</p></div>
        </div>
      </div>

      {analytics && (
        <div className="mt-8">
          <h3 className="text-lg font-bold mb-4">Analytics</h3>
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h4 className="text-sm text-gray-500 mb-2">Browsers</h4>
              <ul className="space-y-1">{Object.entries(analytics.browsers || {}).map(([k, v]) => <li key={k} className="text-sm">{k}: {v}</li>)}</ul>
            </div>
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h4 className="text-sm text-gray-500 mb-2">Devices</h4>
              <ul className="space-y-1">{Object.entries(analytics.devices || {}).map(([k, v]) => <li key={k} className="text-sm">{k}: {v}</li>)}</ul>
            </div>
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h4 className="text-sm text-gray-500 mb-2">Locations</h4>
              <ul className="space-y-1">{Object.entries(analytics.locations || {}).map(([k, v]) => <li key={k} className="text-sm">{k}: {v}</li>)}</ul>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
```

**Step 6: Update App.tsx with full routing**

Replace `web/src/App.tsx`:

```tsx
import { Routes, Route, Navigate } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import Layout from './components/Layout'
import ProtectedRoute from './components/ProtectedRoute'

const Login = lazy(() => import('./pages/Login'))
const Dashboard = lazy(() => import('./pages/Dashboard'))
const UrlList = lazy(() => import('./pages/UrlList'))
const UrlCreate = lazy(() => import('./pages/UrlCreate'))
const UrlDetail = lazy(() => import('./pages/UrlDetail'))

export default function App() {
  return (
    <Suspense fallback={<div className="flex items-center justify-center min-h-screen"><div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full"></div></div>}>
      <Routes>
        <Route path="/admin/login" element={<Login />} />
        <Route path="/admin" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
          <Route index element={<Navigate to="/admin/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="urls" element={<UrlList />} />
          <Route path="urls/create" element={<UrlCreate />} />
          <Route path="urls/:code" element={<UrlDetail />} />
        </Route>
        <Route path="*" element={<Navigate to="/admin/login" replace />} />
      </Routes>
    </Suspense>
  )
}
```

**Step 7: Build and verify**

```bash
cd web && npm run build
cd .. && go build ./cmd/urlshortener/
```

**Step 8: Commit**

```bash
git add web/
git commit -m "feat: add URL management pages to admin SPA"
```

---

### Task 9: Final integration test

**Step 1: Build everything**

```bash
make web-build && make build
```

**Step 2: Run Go tests**

```bash
go test -short ./...
```

**Step 3: Verify the SPA is embedded**

```bash
# The binary should serve /admin/login
./urlshortener &
sleep 2
curl -s http://localhost:8080/admin/login | head -20
curl -s http://localhost:8080/health
kill %1
```

Expected: `/admin/login` returns HTML, `/health` returns JSON.

**Step 4: Commit final state**

```bash
git add -A
git commit -m "chore: final integration verification"
```