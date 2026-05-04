# Error Page & Index Page Design

**Date:** 2026-05-04
**Status:** Approved
**Design system:** monopo saigon (DESIGN.md)

## Overview

Add two new pages to the URL shortener:

1. **Error page** -- renders backend errors as HTML for browser clients, covering redirect errors (400/404/500) and catch-all 404 for unknown routes
2. **Index/home landing page** -- a minimalist presence at `/` promoting the URL shortener, in Bahasa Indonesia

Both pages follow the monopo saigon design system (dark theme, Deep Ocean gradient, glassmorphism cards, Frost White typography).

## Approach

**Approach A: Embedded Go Templates** (chosen)

Two Go templates (`error.html`, `index.html`) embedded via `//go:embed` in the handler package, matching the existing `redirect.html` pattern. No new runtime dependencies. Error messages stay server-controlled (no URL param injection risk).

Alternatives considered:
- **B: Static HTML in `static/`** -- error info in URL params is a security concern; changes the URL users see
- **C: Echo HTML renderer with partials** -- overkill for 3 static-ish pages

---

## 1. Error Page

### Template

`internal/handler/error.html` -- embedded in the handler package, parsed at startup alongside `redirect.html`.

### Go Struct

```go
type ErrorTemplateData struct {
    ErrorCode    int
    ErrorTitle   string
    ErrorMessage string
    ShortCode    string // optional, for 404 contexts
}
```

### Error Scenarios

| Condition | Code | Title (ID) | Message (ID) |
|-----------|------|------------|---------------|
| Short code empty/missing | 400 | Permintaan Tidak Valid | Kode tautan tidak disertakan dalam permintaan. |
| URL not found / expired | 404 | Tautan Tidak Ditemukan | Kode tautan ini tidak ada atau sudah kedaluwarsa. |
| Internal server error | 500 | Terjadi Kesalahan | Server mengalami gangguan. Silakan coba beberapa saat lagi. |
| Catch-all 404 (unknown routes) | 404 | Halaman Tidak Ditemukan | Alamat yang Anda tuju tidak tersedia. |

All user-facing text in Bahasa Indonesia.

### Visual Design

- Monopo saigon dark theme: Midnight Canvas (#000000) background, animated Deep Ocean gradient (blurred, scaled, slowly shifting)
- Glassmorphism card centered on screen with `backdrop-filter: blur(24px)`, inner refraction borders (`border-white/8`, `inset shadow`)
- Large status code (e.g. `404`) rendered at `text-heading-lg` (54px) with Deep Ocean gradient as text fill via `background-clip: text`
- Error title in Frost White (#ffffff), `text-heading-sm` (29px), weight 300
- Error message in Whisper Gray (#6d6d6d), `text-body` (16px), `max-w-[65ch]`
- No navigation or action buttons -- just the error display
- Staggered fade-in animation: code -> title -> message, using `cubic-bezier(0.16, 1, 0.3, 1)` spring physics
- Mobile-responsive: single-column, reduced padding on small screens
- Uses `min-h-[100dvh]` for full viewport height without mobile browser jump issues

### Handler Changes

- **Custom Echo HTTP error handler** in `main.go`: detects `Accept: text/html` header and renders `error.html` instead of returning JSON. Falls back to JSON for API clients.
- **`RedirectURL` modification**: return HTML error pages via template rendering for HTML clients instead of JSON responses.
- **Catch-all route** `/*` (lowest priority, after all other routes): renders 404 error page for HTML clients.

---

## 2. Index / Home Landing Page

### Template

`internal/handler/index.html` -- embedded, no dynamic data required.

### Content (Bahasa Indonesia)

- **Tagline:** "Perpendek. Bagikan. Lacak." (Shorten. Share. Track.)
- **Description:** "Layanan pemendek tautan resmi BPHN untuk menyederhanakan dan melacak tautan dengan aman." (The official BPHN URL shortening service to simplify and track links securely.)
- **Footer:** "BPHN -- Kementerian Hukum dan Hak Asasi Manusia Republik Indonesia"

No CTA button, no dashboard link (for now).

### Visual Design

- Same monopo saigon dark theme and animated gradient background as redirect and error pages
- Sticky header with BPHN logo (white-filtered via CSS)
- Hero section: left-aligned (asymmetric per DESIGN_VARIANCE: 8)
  - Tagline in `text-heading-lg` (54px), Frost White (#ffffff), weight 300 (light)
  - Description in `text-body` (16px), Whisper Gray (#6d6d6d), `max-w-[65ch]`
  - No buttons or CTAs
- Below hero: subtle gradient divider line (`linear-gradient(90deg, transparent, rgba(255,255,255,0.08), transparent)`)
- Footer: institutional attribution text in Whisper Gray, `text-caption` (11px)
- Staggered reveal animations on load (logo -> tagline -> description -> footer) with spring physics
- `min-h-[100dvh]` for full viewport height
- Mobile-responsive: content reflows to single column with `px-4` padding

### Handler Changes

- Add `GET /` route in `main.go` pointing to a new `IndexHandler` method on `URLHandler`
- `IndexHandler` simply renders `index.html` with no template data

---

## 3. Shared Architecture

### Template Embedding

```go
//go:embed redirect.html error.html index.html
var templateFS embed.FS
```

All three templates parsed at startup in a single `template.ParseFS()` call.

### Shared CSS Variables

All three pages (redirect, error, index) share the same `:root` CSS custom properties from DESIGN.md, ensuring visual consistency.

### File Locations

```
internal/handler/
  ├── redirect.html      # existing, redesign with monopo saigon theme
  ├── error.html         # new
  └── index.html         # new
```

### Route Registration (main.go)

```go
e.GET("/", urlHandler.Index)          // NEW: landing page
e.GET("/:code", urlHandler.RedirectURL) // existing
e.GET("/*", catchAllHandler)          // NEW: catch-all 404 (lowest priority)
```

The catch-all must be registered last to avoid overriding specific routes.