package service

import (
	"net"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com", false},
		{"valid with path", "https://example.com/path/to/page", false},
		{"valid with query", "https://example.com?q=test", false},
		{"valid with fragment", "https://example.com/path#section", false},
		{"empty url", "", true},
		{"javascript scheme", "javascript:alert(1)", true},
		{"data scheme", "data:text/html,<script>alert(1)</script>", true},
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com", true},
		{"no scheme", "example.com", true},
		{"localhost", "http://localhost", true},
		{"localhost port", "http://localhost:8080", true},
		{"loopback ip", "http://127.0.0.1", true},
		{"private ip 10", "http://10.0.0.1", true},
		{"private ip 172", "http://172.16.0.1", true},
		{"private ip 192", "http://192.168.1.1", true},
		{"link local", "http://169.254.169.254", true},
		{"too long url", "https://example.com/" + string(make([]byte, 2048)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateShortCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid alphanumeric", "abc123", false},
		{"valid with hyphens", "my-link", false},
		{"valid mixed", "a1b2c3", false},
		{"too short", "ab", true},
		{"too long", "thiscodeiswaytoolongforvalidation", true},
		{"starts with hyphen", "-mylink", true},
		{"ends with hyphen", "mylink-", true},
		{"special chars", "my_link!", true},
		{"spaces", "my link", true},
		{"reserved api", "api", true},
		{"reserved auth", "auth", true},
		{"reserved health", "health", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShortCode(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateShortCode(%q) error = %v, wantErr %v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeDisplayURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"valid https", "https://example.com", "https://example.com"},
		{"valid http", "http://example.com", "http://example.com"},
		{"javascript scheme", "javascript:alert(1)", ""},
		{"data scheme", "data:text/html,<script>", ""},
		{"invalid url", "not-a-url", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeDisplayURL(tt.url)
			if got != tt.want {
				t.Errorf("SanitizeDisplayURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"loopback", "127.0.0.1", true},
		{"private 10", "10.0.0.1", true},
		{"private 172", "172.16.0.1", true},
		{"private 192", "192.168.1.1", true},
		{"link local", "169.254.169.254", true},
		{"public", "8.8.8.8", false},
		{"cloudflare", "1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedIP := parseIP(tt.ip)
			if parsedIP == nil {
				t.Fatalf("failed to parse IP %q", tt.ip)
			}
			got := isPrivateIP(parsedIP)
			if got != tt.want {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func parseIP(s string) net.IP {
	return net.ParseIP(s)
}