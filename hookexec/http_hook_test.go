package hookexec

import (
	"net"
	"testing"
)

func TestCheckSSRF(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"loopback v4", "127.0.0.1", false},
		{"loopback v6", "::1", false},
		{"private 10.x", "10.0.0.1", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"link-local 169.254.x", "169.254.1.1", true},
		{"public", "8.8.8.8", false},
		{"multicast", "224.0.0.1", true},
		{"unspecified", "0.0.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("invalid test IP: %s", tt.ip)
			}
			err := checkSSRF(ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSSRF(%s) error = %v, wantErr = %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

func TestURLMatchesPattern(t *testing.T) {
	tests := []struct {
		url     string
		pattern string
		want    bool
	}{
		{"https://example.com/hook", "https://example.com/*", true},
		{"https://example.com/hook", "https://example.com/hook", true},
		{"https://example.com/hook", "https://other.com/*", false},
		{"https://api.example.com/v1/hook", "https://*.example.com/*", true},
		{"https://example.com", "https://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := urlMatchesPattern(tt.url, tt.pattern)
			if got != tt.want {
				t.Errorf("urlMatchesPattern(%q, %q) = %v, want %v", tt.url, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestCheckURLAllowlist(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		allowed []string
		wantErr bool
	}{
		{"nil allowlist", "https://example.com", nil, false},
		{"empty allowlist", "https://example.com", []string{}, true},
		{"matching", "https://example.com/hook", []string{"https://example.com/*"}, false},
		{"not matching", "https://evil.com/hook", []string{"https://example.com/*"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkURLAllowlist(tt.url, tt.allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkURLAllowlist(%q, %v) error = %v, wantErr = %v", tt.url, tt.allowed, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeHeaderValue(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"normal", "normal"},
		{"with\r\nCRLF", "withCRLF"},
		{"with\x00null", "withnull"},
		{"clean", "clean"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeHeaderValue(tt.input)
			if got != tt.expect {
				t.Errorf("sanitizeHeaderValue(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestInterpolateEnvVars(t *testing.T) {
	// Set test env vars
	t.Setenv("MY_TOKEN", "secret123")
	t.Setenv("OTHER_VAR", "otherval")

	tests := []struct {
		name     string
		input    string
		allowed  []string
		expected string
	}{
		{"allowed var", "Bearer $MY_TOKEN", []string{"MY_TOKEN"}, "Bearer secret123"},
		{"not allowed var", "Bearer $MY_TOKEN", []string{}, "Bearer "},
		{"braced var", "Bearer ${MY_TOKEN}", []string{"MY_TOKEN"}, "Bearer secret123"},
		{"no vars", "plain text", []string{}, "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interpolateEnvVars(tt.input, tt.allowed)
			if got != tt.expected {
				t.Errorf("interpolateEnvVars(%q, %v) = %q, want %q", tt.input, tt.allowed, got, tt.expected)
			}
		})
	}
}
