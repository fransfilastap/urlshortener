# Vite React Frontend & Redirect Fix — Design

**Date**: 2025-05-04
**Status**: Approved

## Problem

1. The redirect page (`static/redirect.html`) is parsed from the filesystem at runtime via `template.ParseFiles("static/redirect.html")`, which fails in Docker/production where the file doesn't exist in the container.
2. There is no admin UI for managing URLs and viewing analytics — all interaction is via API calls.

## Solution

Add a Vite + React + TailwindCSS admin SPA and fix the redirect page to use Go's `//go:embed` directive.

## Architecture

```
Production Request Flow (single Go binary):
┌────────────────────────────────────────────────┐
│  Go Binary                                      │
│                                                 │
│  /:code            → Go renders redirect.html   │
│                     (server-side, no JS needed) │
│                                                 │
│  /api/*            → Echo JSON API handlers     │
│  /auth/login       → Session auth endpoint       │
│  /auth/logout      → Session logout endpoint     │
│  /health           → Health check               │
│  /admin/*          → Serves embedded SPA index  │
│  /assets/*         → Serves embedded JS/CSS     │
└────────────────────────────────────────────────┘
```

- **Redirect page**: Rendered server-side by Go using `html/template`. Data injected directly. Embedded in binary via `//go:embed`. No React dependency.
- **Admin SPA**: Vite + React + TailwindCSS app for URL management, analytics dashboard, and login. Built to static files and embedded in the Go binary.
- **Development**: Vite dev server (`:5173`) proxies API requests to Go server (`:8080`). HMR for rapid frontend development.

## Project Structure

```
urlshortener/
├── web/                              # Vite + React app
│   ├── src/
│   │   ├── App.tsx                   # Router + layout
│   │   ├── main.tsx                  # Entry point
│   │   ├── pages/
│   │   │   ├── Login.tsx             # API key login
│   │   │   ├── Dashboard.tsx         # Overview stats
│   │   │   ├── UrlList.tsx           # List/manage URLs
│   │   │   ├── UrlCreate.tsx         # Create short URL form
│   │   │   ├── UrlDetail.tsx         # URL detail + analytics
│   │   │   └── NotFound.tsx
│   │   ├── components/
│   │   │   ├── Layout.tsx            # Sidebar + header
│   │   │   ├── UrlForm.tsx           # Shared URL create/edit form
│   │   │   ├── AnalyticsChart.tsx    # Click analytics visualization
│   │   │   └── ProtectedRoute.tsx    # Auth guard
│   │   ├── hooks/
│   │   │   └── useAuth.ts            # Auth state + session mgmt
│   │   ├── api/
│   │   │   └── client.ts             # Fetch wrapper with credentials
│   │   └── styles/
│   │       └── index.css             # TailwindCSS entry
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   └── package.json
├── internal/
│   ├── handler/
│   │   ├── url_handler.go            # Updated: embed redirect.html
│   │   ├── spa_handler.go            # NEW: serves embedded SPA
│   │   ├── auth_handler.go           # NEW: login/logout endpoints
│   │   └── middleware.go             # Updated: session auth middleware
│   └── ...
├── static/
│   └── redirect.html                 # Updated: embedded template
├── embed.go                           # Updated: embed web/dist + static
└── Makefile                            # Updated: build frontend step
```

## Redirect Page Fix

**Current**: `template.ParseFiles("static/redirect.html")` — reads from filesystem at runtime.

**Fixed**: Embed the template using `//go:embed` and parse from string:

```go
// internal/handler/redirect.go
//go:embed redirect.html
var redirectTemplate string

func (h *URLHandler) RedirectURL(c echo.Context) error {
    // ...
    tmpl, err := template.New("redirect").Parse(redirectTemplate)
    // ...
}
```

The redirect template remains a Go `html/template` file. Improvements:
- Add `<noscript>` fallback: `<meta http-equiv="refresh" content="0;url={{.OriginalURL}}">`
- Keep existing Tailwind CDN and countdown timer
- Move template file to `internal/handler/redirect.html` alongside its handler

## Authentication — Server-Side Sessions

### Login Flow

1. User visits `/admin/login`
2. Enters API key
3. `POST /auth/login { api_key: "<key>" }`
4. Go validates key, creates session via `gorilla/sessions` cookie store:
   `Set-Cookie: session=<signed-token>; HttpOnly; SameSite=Strict; Path=/`
5. React app redirects to `/admin/dashboard`

### Session Middleware

New `SessionAuthMiddleware` checks session cookie first, falls back to `X-API-Key` header. Both methods work on all authenticated endpoints:

```go
func SessionAuthMiddleware(store *sessions.CookieStore, apiKey string) echo.MiddlewareFunc {
    // 1. Try session cookie → authenticated=true
    // 2. Fall back to X-API-Key header
    // 3. Return 401 if neither works
}
```

### Logout Flow

`DELETE /auth/logout` → Clears session cookie, redirects to `/admin/login`.

### Implementation

- Dependency: `github.com/gorilla/sessions` with cookie store
- `internal/handler/auth_handler.go`: `Login` and `Logout` handlers
- Session stores: `authenticated: bool`, `api_key: string`
- React: `credentials: 'include'` on all fetch calls
- `useAuth` hook: checks `/auth/me` on load, redirects to login if unauthenticated

## Admin SPA — Key Dependencies

- **React 19** + **React Router v7**
- **TailwindCSS v4** (via Vite plugin)
- **Vite 6** for build tooling
- **Recharts** or **Chart.js** for analytics visualization
- No Redux — simple `useAuth` hook + `fetch` wrapper

## Build & Deployment

### Makefile Additions

```makefile
.PHONY: web-deps
web-deps:
	cd web && npm install

.PHONY: web-build
web-build:
	cd web && npm run build

.PHONY: web-dev
web-dev:
	cd web && npm run dev

# Build: frontend first, then Go
build: web-build
	go build -o urlshortener ./cmd/urlshortener
```

### Dockerfile Update

```dockerfile
# Build frontend
FROM node:20-alpine AS web-builder
WORKDIR /app/web
COPY web/ .
RUN npm install && npm run build

# Build Go binary (copy web/dist from previous stage)
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
CMD ["./urlshortener"]
```

### Development Workflow

```bash
# Terminal 1: Go server
make run

# Terminal 2: Vite dev server (proxies API to :8080)
make web-dev
```

Vite config proxies `/api/*` and `/auth/*` to `http://localhost:8080`.

## embed.go Update

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

## SPA Handler

```go
// internal/handler/spa_handler.go
func RegisterSPA(e *echo.Echo, distFS embed.FS) {
    // Serve /assets/* from embedded dist
    // Serve /admin/* → index.html (client-side routing)
    // Service worker / vite manifest handling
}
```

## Scope

Included:
- Fix redirect.html embedding
- Vite + React + TailwindCSS admin SPA
- Server-side session auth (login/logout)
- Dual auth middleware (session cookie + API key header)
- SPA handler for serving embedded frontend
- Build pipeline (Makefile + Dockerfile)
- Login, Dashboard, URL List, URL Create, URL Detail pages

Out of scope (future work):
- OAuth / social login
- User registration / multi-user auth
- Real-time analytics (WebSocket)
- URL preview (Open Graph metadata fetching)
- Branded short link domains