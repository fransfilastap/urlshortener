# Error Page & Index Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an HTML error page for redirect errors and catch-all 404s, plus a minimalist landing page at `/`, both styled with the monopo saigon design system.

**Architecture:** Two new Go HTML templates (`error.html`, `index.html`) embedded in the handler package alongside the existing `redirect.html`. A custom Echo HTTP error handler renders `error.html` for browser requests (returns JSON for API clients). A new `Index` handler renders the static `index.html` at `GET /`. The redirect handler's error paths are updated to render the error template instead of returning JSON for HTML clients.

**Tech Stack:** Go html/template, Echo v4, hand-written CSS (no Tailwind CDN on new pages), monopo saigon design tokens

---

### Task 1: Create the error page template

**Files:**
- Create: `internal/handler/error.html`

**Step 1: Create `internal/handler/error.html`**

Create the error page template. It follows the monopo saigon design system (dark theme, Deep Ocean gradient background, glassmorphism card). All text in Bahasa Indonesia. Template receives `ErrorTemplateData` with `ErrorCode`, `ErrorTitle`, `ErrorMessage`, and `ShortCode` fields.

```html
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
    <title>{{.ErrorCode}} — {{.ErrorTitle}}</title>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&display=swap');

        :root {
            --color-midnight-canvas: #000000;
            --color-frost-white: #ffffff;
            --color-deep-shadow: #181818;
            --color-whisper-gray: #6d6d6d;
            --color-misty-gray: #636363;
            --gradient-deep-ocean: linear-gradient(135deg, rgb(160, 224, 171) 0%, rgb(255, 172, 46) 50%, rgb(165, 45, 37) 100%);
            --gradient-deep-ocean-subtle: linear-gradient(135deg, rgba(160, 224, 171, 0.15) 0%, rgba(255, 172, 46, 0.08) 50%, rgba(165, 45, 37, 0.12) 100%);
            --font-roobert: 'Inter', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            --text-caption: 11px;
            --text-body: 16px;
            --text-heading-sm: 29px;
            --text-heading: 39px;
            --spacing-8: 8px;
            --spacing-12: 12px;
            --spacing-28: 28px;
            --spacing-40: 40px;
            --radius-cards: 10px;
        }

        *, *::before, *::after {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        html {
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }

        body {
            font-family: var(--font-roobert);
            background: var(--color-midnight-canvas);
            color: var(--color-frost-white);
            min-height: 100dvh;
            display: flex;
            align-items: center;
            justify-content: center;
            overflow-x: hidden;
        }

        .gradient-bg {
            position: fixed;
            inset: 0;
            z-index: 0;
            background: var(--gradient-deep-ocean);
            opacity: 0.35;
            filter: blur(80px) saturate(1.2);
            transform: scale(1.4);
            animation: gradientShift 12s ease-in-out infinite alternate;
        }

        .gradient-bg::after {
            content: '';
            position: absolute;
            inset: 0;
            background: radial-gradient(ellipse at 30% 20%, rgba(160, 224, 171, 0.25) 0%, transparent 50%),
                        radial-gradient(ellipse at 70% 80%, rgba(165, 45, 37, 0.2) 0%, transparent 50%);
            animation: orbFloat 8s ease-in-out infinite alternate;
        }

        @keyframes gradientShift {
            0% { transform: scale(1.4) translate(0, 0); }
            100% { transform: scale(1.5) translate(-2%, 3%); }
        }

        @keyframes orbFloat {
            0% { opacity: 0.6; }
            100% { opacity: 1; }
        }

        .error-card {
            position: relative;
            z-index: 10;
            width: 100%;
            max-width: 520px;
            background: rgba(255, 255, 255, 0.04);
            border: 1px solid rgba(255, 255, 255, 0.08);
            border-radius: var(--radius-cards);
            padding: var(--spacing-40);
            margin: var(--spacing-28);
            backdrop-filter: blur(24px);
            -webkit-backdrop-filter: blur(24px);
            box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.06),
                        0 20px 60px -15px rgba(0, 0, 0, 0.5);
            animation: cardReveal 0.8s cubic-bezier(0.16, 1, 0.3, 1) forwards;
            opacity: 0;
            transform: translateY(20px);
        }

        @keyframes cardReveal {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .error-code {
            font-size: 72px;
            font-weight: 300;
            line-height: 1;
            background: var(--gradient-deep-ocean);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: var(--spacing-28);
            animation: codeReveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.1s forwards;
            opacity: 0;
            transform: translateY(12px);
        }

        @keyframes codeReveal {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .error-title {
            font-size: var(--text-heading-sm);
            font-weight: 400;
            line-height: 1.21;
            color: var(--color-frost-white);
            margin-bottom: var(--spacing-12);
            animation: titleReveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.25s forwards;
            opacity: 0;
            transform: translateY(12px);
        }

        @keyframes titleReveal {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .error-message {
            font-size: var(--text-body);
            line-height: 1.5;
            color: var(--color-whisper-gray);
            max-width: 55ch;
            animation: msgReveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.4s forwards;
            opacity: 0;
            transform: translateY(12px);
        }

        @keyframes msgReveal {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .divider-line {
            width: 100%;
            height: 1px;
            background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.08), transparent);
            margin: var(--spacing-28) 0;
            animation: msgReveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.55s forwards;
            opacity: 0;
        }

        .error-meta {
            font-size: var(--text-caption);
            color: var(--color-whisper-gray);
            opacity: 0.5;
            animation: msgReveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.65s forwards;
            opacity: 0;
        }

        @media (max-width: 640px) {
            .error-card {
                padding: var(--spacing-28);
            }
            .error-code {
                font-size: 56px;
            }
        }
    </style>
</head>
<body>
    <div class="gradient-bg" aria-hidden="true"></div>

    <div class="error-card" role="alert">
        <div class="error-code">{{.ErrorCode}}</div>
        <h1 class="error-title">{{.ErrorTitle}}</h1>
        <p class="error-message">{{.ErrorMessage}}</p>
        <div class="divider-line" aria-hidden="true"></div>
        <div class="error-meta">s.bphn.go.id</div>
    </div>
</body>
</html>
```

**Step 2: Verify the template file exists**

Run: `ls -la internal/handler/error.html`
Expected: File exists with content matching the above

**Step 3: Commit**

```bash
git add internal/handler/error.html
git commit -m "feat: add error page template (monopo saigon design)"
```

---

### Task 2: Create the index/home page template

**Files:**
- Create: `internal/handler/index.html`

**Step 1: Create `internal/handler/index.html`**

Create the landing page template. Minimalist presence: BPHN logo, Bahasa Indonesia tagline, brief service description, institutional footer. Monopo saigon dark theme. No dynamic data needed.

```html
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
    <title>BPHN URL Shortener</title>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&display=swap');

        :root {
            --color-midnight-canvas: #000000;
            --color-frost-white: #ffffff;
            --color-deep-shadow: #181818;
            --color-whisper-gray: #6d6d6d;
            --color-misty-gray: #636363;
            --gradient-deep-ocean: linear-gradient(135deg, rgb(160, 224, 171) 0%, rgb(255, 172, 46) 50%, rgb(165, 45, 37) 100%);
            --gradient-deep-ocean-subtle: linear-gradient(135deg, rgba(160, 224, 171, 0.15) 0%, rgba(255, 172, 46, 0.08) 50%, rgba(165, 45, 37, 0.12) 100%);
            --font-roobert: 'Inter', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            --text-caption: 11px;
            --text-body: 16px;
            --text-subheading: 18px;
            --text-heading: 39px;
            --text-heading-lg: 54px;
            --spacing-8: 8px;
            --spacing-12: 12px;
            --spacing-28: 28px;
            --spacing-40: 40px;
            --spacing-48: 48px;
            --spacing-64: 64px;
            --spacing-152: 152px;
            --page-max-width: 1078px;
            --radius-cards: 10px;
            --radius-buttons: 75px;
        }

        *, *::before, *::after {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        html {
            -webkit-font-smoothing: antialiased;
            -moz-osx-font-smoothing: grayscale;
        }

        body {
            font-family: var(--font-roobert);
            background: var(--color-midnight-canvas);
            color: var(--color-frost-white);
            min-height: 100dvh;
            display: flex;
            flex-direction: column;
            overflow-x: hidden;
        }

        .gradient-bg {
            position: fixed;
            inset: 0;
            z-index: 0;
            background: var(--gradient-deep-ocean);
            opacity: 0.35;
            filter: blur(80px) saturate(1.2);
            transform: scale(1.4);
            animation: gradientShift 12s ease-in-out infinite alternate;
        }

        .gradient-bg::after {
            content: '';
            position: absolute;
            inset: 0;
            background: radial-gradient(ellipse at 30% 20%, rgba(160, 224, 171, 0.25) 0%, transparent 50%),
                        radial-gradient(ellipse at 70% 80%, rgba(165, 45, 37, 0.2) 0%, transparent 50%);
            animation: orbFloat 8s ease-in-out infinite alternate;
        }

        @keyframes gradientShift {
            0% { transform: scale(1.4) translate(0, 0); }
            100% { transform: scale(1.5) translate(-2%, 3%); }
        }

        @keyframes orbFloat {
            0% { opacity: 0.6; }
            100% { opacity: 1; }
        }

        header {
            position: relative;
            z-index: 10;
            display: flex;
            align-items: center;
            padding: var(--spacing-12) var(--spacing-28);
            border-bottom: 1px solid rgba(255, 255, 255, 0.08);
            backdrop-filter: blur(12px);
            -webkit-backdrop-filter: blur(12px);
        }

        header img {
            height: 36px;
            filter: brightness(0) invert(1);
            opacity: 0.9;
            animation: logoReveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
            opacity: 0;
        }

        @keyframes logoReveal {
            to { opacity: 0.9; }
        }

        main {
            position: relative;
            z-index: 10;
            flex: 1;
            display: flex;
            align-items: center;
            padding: var(--spacing-64) var(--spacing-28);
        }

        .hero-content {
            max-width: var(--page-max-width);
            width: 100%;
            margin: 0 auto;
        }

        .tagline {
            font-size: var(--text-heading-lg);
            font-weight: 300;
            line-height: 1.39;
            color: var(--color-frost-white);
            margin-bottom: var(--spacing-28);
            letter-spacing: -0.02em;
            animation: taglineReveal 0.8s cubic-bezier(0.16, 1, 0.3, 1) 0.15s forwards;
            opacity: 0;
            transform: translateY(24px);
        }

        @keyframes taglineReveal {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .tagline .accent {
            background: var(--gradient-deep-ocean);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .description {
            font-size: var(--text-body);
            line-height: 1.5;
            color: var(--color-whisper-gray);
            max-width: 55ch;
            margin-bottom: var(--spacing-48);
            animation: descReveal 0.8s cubic-bezier(0.16, 1, 0.3, 1) 0.3s forwards;
            opacity: 0;
            transform: translateY(20px);
        }

        @keyframes descReveal {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .features {
            display: grid;
            grid-template-columns: 1fr;
            gap: var(--spacing-28);
            max-width: 560px;
            animation: featuresReveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) 0.45s forwards;
            opacity: 0;
            transform: translateY(16px);
        }

        @keyframes featuresReveal {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .feature-item {
            display: flex;
            align-items: flex-start;
            gap: var(--spacing-12);
        }

        .feature-icon {
            width: 32px;
            height: 32px;
            border-radius: 6px;
            background: rgba(255, 255, 255, 0.04);
            border: 1px solid rgba(255, 255, 255, 0.08);
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
            margin-top: 2px;
        }

        .feature-icon svg {
            width: 16px;
            height: 16px;
            stroke: var(--color-whisper-gray);
        }

        .feature-text h3 {
            font-size: var(--text-subheading);
            font-weight: 400;
            color: var(--color-frost-white);
            margin-bottom: 2px;
        }

        .feature-text p {
            font-size: var(--text-caption);
            line-height: 1.58;
            color: var(--color-whisper-gray);
        }

        footer {
            position: relative;
            z-index: 10;
            padding: var(--spacing-12) var(--spacing-28);
            text-align: center;
        }

        footer span {
            font-size: var(--text-caption);
            color: var(--color-whisper-gray);
            opacity: 0.5;
        }

        @media (max-width: 640px) {
            header {
                padding: var(--spacing-12) var(--spacing-12);
            }
            main {
                padding: var(--spacing-40) var(--spacing-12);
            }
            .tagline {
                font-size: var(--text-heading);
            }
            footer {
                padding: var(--spacing-12) var(--spacing-12);
            }
        }
    </style>
</head>
<body>
    <div class="gradient-bg" aria-hidden="true"></div>

    <header>
        <img src="/static/img/logo.svg" alt="BPHN" />
    </header>

    <main>
        <div class="hero-content">
            <h1 class="tagline">
                <span class="accent">Perpendek.</span> Bagikan. Lacak.
            </h1>
            <p class="description">
                Layanan pemendek tautan resmi BPHN untuk menyederhanakan dan melacak tautan dengan aman.
            </p>

            <div class="features">
                <div class="feature-item">
                    <div class="feature-icon">
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
                    </div>
                    <div class="feature-text">
                        <h3>Pengalihan Cepat</h3>
                        <p>Redirect instan tanpa latensi yang terasa oleh pengguna akhir.</p>
                    </div>
                </div>
                <div class="feature-item">
                    <div class="feature-icon">
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20V10"/><path d="M18 20V4"/><path d="M6 20v-4"/></svg>
                    </div>
                    <div class="feature-text">
                        <h3>Analitik Tautan</h3>
                        <p>Pantau performa tautan secara real-time dengan data klik dan perangkat.</p>
                    </div>
                </div>
                <div class="feature-item">
                    <div class="feature-icon">
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
                    </div>
                    <div class="feature-text">
                        <h3>Aman dan Terlacak</h3>
                        <p>Setiap tautan terverifikasi dan dapat dilacak untuk transparansi penuh.</p>
                    </div>
                </div>
            </div>
        </div>
    </main>

    <footer>
        <span>BPHN — Kementerian Hukum dan Hak Asasi Manusia Republik Indonesia</span>
    </footer>
</body>
</html>
```

**Step 2: Verify the template file exists**

Run: `ls -la internal/handler/index.html`
Expected: File exists with content matching the above

**Step 3: Commit**

```bash
git add internal/handler/index.html
git commit -m "feat: add index/landing page template (monopo saigon design, Bahasa)"
```

---

### Task 3: Update the handler to embed and parse all templates

**Files:**
- Modify: `internal/handler/url_handler.go:1-21` (embed directive and template parsing)

**Step 1: Update the embed directive and template parsing in `url_handler.go`**

Replace the current embed and template parsing block (lines 17-20):

```go
//go:embed redirect.html
var redirectHTML string

var redirectTmpl = template.Must(template.New("redirect").Parse(redirectHTML))
```

With:

```go
//go:embed redirect.html error.html index.html
var templateFS string

var (
	redirectTmpl = template.Must(template.New("redirect").ParseFS(template.Must(template.New("").Parse(templateFS)), "redirect.html"))
	errorTmpl    = template.Must(template.New("error").ParseFS(template.Must(template.New("").Parse(templateFS)), "error.html"))
	indexTmpl    = template.Must(template.New("index").ParseFS(template.Must(template.New("").Parse(templateFS)), "index.html"))
)
```

Wait — this approach is fragile with `ParseFS` on a string. The better approach is to keep using `//go:embed` for individual files which works with `template.New().Parse()`. Let me use the simpler, proven pattern:

```go
//go:embed redirect.html error.html index.html
var (
	redirectHTML string
	errorHTML    string
	indexHTML    string
)

var (
	redirectTmpl = template.Must(template.New("redirect").Parse(redirectHTML))
	errorTmpl     = template.Must(template.New("error").Parse(errorHTML))
	indexTmpl     = template.Must(template.New("index").Parse(indexHTML))
)
```

This uses Go's `//go:embed` with multiple variable names in a single directive, each mapped to a file with the same name. Each template is parsed independently.

**Step 2: Verify it compiles**

Run: `go build ./cmd/urlshortener/`
Expected: Build succeeds with no errors

**Step 3: Commit**

```bash
git add internal/handler/url_handler.go
git commit -m "refactor: embed error and index templates alongside redirect template"
```

---

### Task 4: Add ErrorTemplateData struct and helper function

**Files:**
- Modify: `internal/handler/url_handler.go` (add struct and render helper)

**Step 1: Add `ErrorTemplateData` struct after `TemplateData`**

After the existing `TemplateData` struct (around line 198 in the `RedirectURL` function), extract it and the new error struct to the package level. Add this after the `URLHandler` struct definition:

```go
type TemplateData struct {
	OriginalURL string
	ShortURL    string
	Clicks      int64
}

type ErrorTemplateData struct {
	ErrorCode    int
	ErrorTitle   string
	ErrorMessage string
	ShortCode    string
}
```

**Step 2: Remove the inline `TemplateData` definition inside `RedirectURL`**

In `RedirectURL`, the `TemplateData` struct is defined locally inside the function (lines 197-201). Remove the local definition since we've moved it to package level.

Change:
```go
		type TemplateData struct {
			OriginalURL string
			ShortURL    string
			Clicks      int64
		}

		data := TemplateData{
```

To:
```go
		data := TemplateData{
```

**Step 3: Add `renderErrorPage` helper function**

Add this function at the end of `url_handler.go`:

```go
func renderErrorHTML(c echo.Context, code int, title string, message string) error {
	data := ErrorTemplateData{
		ErrorCode:    code,
		ErrorTitle:   title,
		ErrorMessage: message,
		ShortCode:    c.Param("code"),
	}
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(code)
	if err := errorTmpl.Execute(c.Response().Writer, data); err != nil {
		log.Error().Err(err).Msg("Failed to render error page")
		return c.JSON(code, map[string]string{"error": title})
	}
	return nil
}
```

**Step 4: Verify it compiles**

Run: `go build ./cmd/urlshortener/`
Expected: Build succeeds with no errors

**Step 5: Commit**

```bash
git add internal/handler/url_handler.go
git commit -m "feat: add ErrorTemplateData struct and renderErrorHTML helper"
```

---

### Task 5: Update RedirectURL to render HTML errors for browser clients

**Files:**
- Modify: `internal/handler/url_handler.go` (the `RedirectURL` function)

**Step 1: Update the error handling in `RedirectURL`**

In the `RedirectURL` function, after extracting the code param (line 127-131), replace the JSON error returns with HTML-aware error handling. The pattern: if client accepts `text/html`, render the error template; otherwise return JSON.

Replace the entire error handling block in `RedirectURL` (lines 128-143):

```go
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
```

With:

```go
	code := c.Param("code")
	acceptsHTML := strings.Contains(c.Request().Header.Get("Accept"), "text/html")

	if code == "" {
		log.Error().Msg("Missing URL code in redirect request")
		if acceptsHTML {
			return renderErrorHTML(c, http.StatusBadRequest, "Permintaan Tidak Valid", "Kode tautan tidak disertakan dalam permintaan.")
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing URL code"})
	}

	log.Debug().Str("code", code).Msg("Redirecting short URL")

	url, err := h.service.GetByShort(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			log.Error().Err(err).Str("code", code).Msg("URL not found for redirect")
			if acceptsHTML {
				return renderErrorHTML(c, http.StatusNotFound, "Tautan Tidak Ditemukan", "Kode tautan ini tidak ada atau sudah kedaluwarsa.")
			}
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		log.Error().Err(err).Str("code", code).Msg("Failed to retrieve URL for redirect")
		if acceptsHTML {
			return renderErrorHTML(c, http.StatusInternalServerError, "Terjadi Kesalahan", "Server mengalami gangguan. Silakan coba beberapa saat lagi.")
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}
```

**Step 2: Verify it compiles**

Run: `go build ./cmd/urlshortener/`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/handler/url_handler.go
git commit -m "feat: redirect handler returns HTML error pages for browser clients"
```

---

### Task 6: Add Index handler and register GET / route

**Files:**
- Modify: `internal/handler/url_handler.go` (add Index method)
- Modify: `cmd/urlshortener/main.go` (register route)

**Step 1: Add `Index` method to `URLHandler`**

Add this method to `url_handler.go` after the `RedirectURL` method:

```go
func (h *URLHandler) Index(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(http.StatusOK)
	if err := indexTmpl.Execute(c.Response().Writer, nil); err != nil {
		log.Error().Err(err).Msg("Failed to render index page")
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	return nil
}
```

**Step 2: Register the `GET /` route in `main.go`**

In `cmd/urlshortener/main.go`, add the index route before the `/:code` route (around line 129). Insert before the `// Public redirect endpoint` comment:

```go
	// Index / landing page
	e.GET("/", urlHandler.Index)

	// Public redirect endpoint
```

**Step 3: Verify it compiles**

Run: `go build ./cmd/urlshortener/`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/handler/url_handler.go cmd/urlshortener/main.go
git commit -m "feat: add Index handler and register GET / route"
```

---

### Task 7: Add catch-all 404 handler and custom Echo error handler

**Files:**
- Modify: `cmd/urlshortener/main.go` (add custom error handler and catch-all route)

**Step 1: Add custom Echo error handler in `main.go`**

Create a new file `internal/handler/error_handler.go` to keep the error handler logic clean:

```go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func CustomHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := echo.ErrInternalServerError.Code
	message := "Internal Server Error"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		message = he.Message.(string)
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
			ErrorMessage:  detail,
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
```

Wait, this file needs `strings` import. Let me write the complete file.

**Step 2: Create `internal/handler/error_handler.go`**

```go
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
			ErrorCode:     code,
			ErrorTitle:    title,
			ErrorMessage:  detail,
			ShortCode:     "",
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
```

**Step 3: Register the custom error handler in `main.go`**

In `cmd/urlshortener/main.go`, after creating the Echo instance `e := echo.New()`, add:

```go
	e.HTTPErrorHandler = handler.CustomHTTPErrorHandler
```

This should go right after `e := echo.New()` (line 116) and before the middleware registrations.

**Step 4: Verify it compiles**

Run: `go build ./cmd/urlshortener/`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add internal/handler/error_handler.go cmd/urlshortener/main.go
git commit -m "feat: add custom HTTP error handler for HTML/JSON content negotiation"
```

---

### Task 8: Update existing tests for HTML-aware RedirectURL

**Files:**
- Modify: `internal/handler/url_handler_test.go`

**Step 1: Update `TestRedirectURL` tests to account for HTML error pages**

The existing `TestRedirectURL` tests in `url_handler_test.go` use the `TestURLHandler` which is a simplified clone. However, the real `URLHandler.RedirectURL` now checks `Accept: text/html`. The new tests should verify both JSON and HTML response paths.

Add new tests after the existing `TestRedirectURL` block (after line 357) in `url_handler_test.go`:

```go
func TestRedirectURL_HTMLNotFound(t *testing.T) {
	e := echo.New()
	mockService := new(MockURLService)
	h := NewURLHandler(mockService, "http://localhost:8080", "test-api-key")

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues("notfound")

	mockService.On("GetByShort", mock.Anything, "notfound").Return(nil, domain.ErrURLNotFound)

	err := h.RedirectURL(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "Tautan Tidak Ditemukan")
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")

	mockService.AssertExpectations(t)
}

func TestRedirectURL_JSONNotFound(t *testing.T) {
	e := echo.New()
	mockService := new(MockURLService)
	h := NewURLHandler(mockService, "http://localhost:8080", "test-api-key")

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues("notfound")

	mockService.On("GetByShort", mock.Anything, "notfound").Return(nil, domain.ErrURLNotFound)

	err := h.RedirectURL(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	var response map[string]string
	jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, jsonErr)
	assert.Equal(t, "URL not found", response["error"])

	mockService.AssertExpectations(t)
}
```

**Step 2: Run the tests**

Run: `go test ./internal/handler/ -v -run TestRedirectURL`
Expected: Both new tests pass. Note: the existing `TestRedirectURL` tests use `TestURLHandler` which is a simplified mock that doesn't have the HTML logic, so they should still pass as-is.

**Step 3: Commit**

```bash
git add internal/handler/url_handler_test.go
git commit -m "test: add HTML and JSON content negotiation tests for redirect errors"
```

---

### Task 9: Add test for Index handler

**Files:**
- Modify: `internal/handler/url_handler_test.go`

**Step 1: Add Index handler test**

```go
func TestIndexHandler(t *testing.T) {
	e := echo.New()
	mockService := new(MockURLService)
	h := NewURLHandler(mockService, "http://localhost:8080", "test-api-key")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Index(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Perpendek")
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}
```

**Step 2: Run the test**

Run: `go test ./internal/handler/ -v -run TestIndexHandler`
Expected: Test passes

**Step 3: Commit**

```bash
git add internal/handler/url_handler_test.go
git commit -m "test: add Index handler test"
```

---

### Task 10: Run full test suite and final build

**Files:**
- No changes — verification only

**Step 1: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 2: Build the binary**

Run: `go build -o /tmp/urlshortener ./cmd/urlshortener/`
Expected: Build succeeds

**Step 3: Final commit (if any unstaged changes)**

```bash
git status
```

If everything is clean, no commit needed. If there are changes, commit them with an appropriate message.