package security

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

// TestClientIP covers 006 S-4: the IP is the TCP peer of RemoteAddr without
// the port, and a forwarding header never changes it.
func TestClientIP(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		name       string
		remoteAddr string
		want       string // "" means no IP
	}{
		{"ipv4", "203.0.113.7:4444", "203.0.113.7"},
		{"ipv6 loopback", "[::1]:8443", "::1"},
		{"ipv6", "[2001:db8::1]:50000", "2001:db8::1"},
		{"ipv6 with a zone", "[fe80::1%eth0]:443", "fe80::1"},
		{"ipv4-mapped ipv6", "[::ffff:192.0.2.10]:1234", "192.0.2.10"},
		{"no port", "203.0.113.7", ""},
		{"empty", "", ""},
		{"hostname", "localhost:443", ""},
		{"garbage", "not an address", ""},
		{"port out of range", "203.0.113.7:99999", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequestWithContext(t.Context(), "POST", "/api/login", nil)
			r.RemoteAddr = tt.remoteAddr
			r.Header.Set("X-Forwarded-For", "198.51.100.66")
			r.Header.Set("X-Real-IP", "198.51.100.77")
			r.Header.Set("Forwarded", "for=198.51.100.88")

			ip, ok := s.ClientIP(r)
			if tt.want == "" {
				if ok || ip.IsValid() {
					t.Errorf("ClientIP(%q) = %v, %v; want no IP", tt.remoteAddr, ip, ok)
				}
				return
			}
			if !ok || ip.String() != tt.want {
				t.Errorf("ClientIP(%q) = %v, %v; want %s", tt.remoteAddr, ip, ok, tt.want)
			}
		})
	}
}

func TestParseAuditType(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	for _, typ := range domain.AuditEventTypes() {
		if got, err := s.ParseAuditType(string(typ)); err != nil || got != typ {
			t.Errorf("ParseAuditType(%q) = %q, %v", typ, got, err)
		}
	}
	if got, err := s.ParseAuditType(""); err != nil || got != "" {
		t.Errorf(`ParseAuditType("") = %q, %v; want no filter`, got, err)
	}
	for _, raw := range []string{"LOGIN_FAILED", "login_failed ", "post", "x", "login_succeeded,logout"} {
		if _, err := s.ParseAuditType(raw); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("ParseAuditType(%q) = %v, want ErrInvalidInput", raw, err)
		}
	}
}

// TestValidateRemovalReason covers the reason rules of 006 D-2: like a post
// body, with a limit of 500 code points.
func TestValidateRemovalReason(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		name   string
		reason string
		want   string
		valid  bool
	}{
		{"simple", "Spam", "Spam", true},
		{"trimmed", "  Ofensivo \n", "Ofensivo", true},
		{"500", strings.Repeat("a", 500), strings.Repeat("a", 500), true},
		{"500 multibyte", strings.Repeat("ã", 500), strings.Repeat("ã", 500), true},
		{"501", strings.Repeat("a", 501), "", false},
		{"line breaks and tabs", "linha 1\r\nlinha 2\tfim", "linha 1\r\nlinha 2\tfim", true},
		{"empty", "", "", false},
		{"whitespace only", " \n\t ", "", false},
		{"NUL", "spam\x00", "", false},
		{"escape", "spam\x1b[31m", "", false},
		{"invalid utf-8", "spam\xff", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := s.ValidateRemovalReason(tt.reason)
			if !tt.valid {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Errorf("error = %v, want ErrInvalidInput", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("ValidateRemovalReason = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}
