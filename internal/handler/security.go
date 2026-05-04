package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type visitor struct {
	count    int
	lastSeen time.Time
}

type RateLimiter struct {
	visitors sync.Map
	rate    int
	window  time.Duration
	mu      sync.Mutex
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		rate:   rate,
		window: window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		rl.visitors.Range(func(key, value interface{}) bool {
			v := value.(*visitor)
			if now.Sub(v.lastSeen) > rl.window {
				rl.visitors.Delete(key)
			}
			return true
		})
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	now := time.Now()
	val, ok := rl.visitors.Load(ip)
	if !ok {
		rl.visitors.Store(ip, &visitor{count: 1, lastSeen: now})
		return true
	}

	v := val.(*visitor)
	if now.Sub(v.lastSeen) > rl.window {
		v.count = 1
		v.lastSeen = now
		return true
	}

	v.count++
	v.lastSeen = now
	return v.count <= rl.rate
}

func RateLimitMiddleware(rl *RateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !rl.Allow(ip) {
				return c.JSON(429, map[string]string{"error": "Too many requests"})
			}
			return next(c)
		}
	}
}

func MaskIP(ip string) string {
	if ip == "" {
		return ""
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		host, _, err := net.SplitHostPort(ip)
		if err != nil {
			return "***"
		}
		parsedIP = net.ParseIP(host)
		if parsedIP == nil {
			return "***"
		}
		ip = host
	}

	if parsedIP.To4() != nil {
		parts := strings.Split(ip, ".")
		if len(parts) == 4 {
			return parts[0] + "." + parts[1] + "." + "*.*"
		}
	}

	if parsedIP.To16() != nil {
		parts := strings.Split(ip, ":")
		if len(parts) >= 4 {
			return strings.Join(parts[:3], ":") + ":*"
		}
	}

	return "***"
}

func HashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:])[:16]
}

func SanitizeDisplayURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return parsed.String()
}

func SecurityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "0")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			c.Response().Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			c.Response().Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://fonts.gstatic.com; font-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
			return next(c)
		}
	}
}