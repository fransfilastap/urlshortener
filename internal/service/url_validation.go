package service

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/fransfilastap/urlshortener/internal/domain"
)

var allowedSchemes = map[string]bool{"http": true, "https": true}

var shortCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]{1,18}[a-zA-Z0-9]$`)

func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return domain.ErrInvalidURL
	}

	if len(rawURL) > 2048 {
		return fmt.Errorf("%w: URL exceeds maximum length", domain.ErrInvalidURL)
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidURL, err)
	}

	if !allowedSchemes[parsed.Scheme] {
		return fmt.Errorf("%w: scheme %q is not allowed; use http or https", domain.ErrInvalidURL, parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%w: host is required", domain.ErrInvalidURL)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("%w: hostname is required", domain.ErrInvalidURL)
	}

	if err := validateHost(hostname); err != nil {
		return err
	}

	return nil
}

func validateHost(hostname string) error {
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") {
		return fmt.Errorf("%w: localhost is not allowed", domain.ErrInvalidURL)
	}

	ips := resolveHost(hostname)
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("%w: private or reserved IP addresses are not allowed", domain.ErrInvalidURL)
		}
	}

	return nil
}

func resolveHost(hostname string) []net.IP {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil
	}
	return ips
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{parseCIDR("10.0.0.0/8")},
		{parseCIDR("172.16.0.0/12")},
		{parseCIDR("192.168.0.0/16")},
		{parseCIDR("127.0.0.0/8")},
		{parseCIDR("169.254.0.0/16")},
		{parseCIDR("::1/128")},
		{parseCIDR("fc00::/7")},
		{parseCIDR("fe80::/10")},
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		return &net.IPNet{}
	}
	return network
}

func ValidateShortCode(code string) error {
	if code == "" {
		return fmt.Errorf("%w: short code is required", domain.ErrInvalidURL)
	}

	if len(code) < 3 || len(code) > 20 {
		return fmt.Errorf("%w: short code must be between 3 and 20 characters", domain.ErrInvalidURL)
	}

	if !shortCodeRegex.MatchString(code) {
		return fmt.Errorf("%w: short code must contain only letters, numbers, and hyphens (cannot start or end with a hyphen)", domain.ErrInvalidURL)
	}

	reserved := map[string]bool{
		"api": true, "auth": true, "static": true, "admin": true, "health": true,
		"assets": true, "vite": true,
	}
	if reserved[strings.ToLower(code)] {
		return fmt.Errorf("%w: short code %q is reserved", domain.ErrInvalidURL, code)
	}

	return nil
}

func SanitizeDisplayURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if !allowedSchemes[parsed.Scheme] {
		return ""
	}
	return parsed.String()
}