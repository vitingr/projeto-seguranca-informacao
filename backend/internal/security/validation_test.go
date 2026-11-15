package security

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

// sentinelPassword marks the password in request bodies, so tests can check
// that it never leaks into errors or logs.
const sentinelPassword = "Sentinel-Pa55word-7f3a"

type loginBody struct {
	Login string `json:"login"`
	Senha string `json:"senha"`
}

func newJSONRequest(t *testing.T, contentType, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/login", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req
}

// padTo returns body followed by spaces up to n bytes.
func padTo(body string, n int) string {
	return body + strings.Repeat(" ", n-len(body))
}

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	const ct = "application/json"
	valid := `{"login":"admin","senha":"` + sentinelPassword + `"}`

	tests := []struct {
		name        string
		contentType string
		body        string
		wantErr     string // reason in the error; empty = success
	}{
		{"valid", ct, valid, ""},
		{"charset utf-8", "application/json; charset=utf-8", valid, ""},
		{"charset UTF-8 upper case", "Application/JSON; charset=UTF-8", valid, ""},
		{"trailing whitespace", ct, valid + " \n\t", ""},
		{"exactly the limit", ct, padTo(valid, MaxBodyBytes), ""},
		{"missing content type", "", valid, reasonContentType},
		{"text/plain", "text/plain", valid, reasonContentType},
		{"json suffix type", "application/problem+json", valid, reasonContentType},
		{"other charset", "application/json; charset=latin1", valid, reasonContentType},
		{"malformed content type", "application/json; charset", valid, reasonContentType},
		{"one byte over the limit", ct, padTo(valid, MaxBodyBytes+1), reasonTooLarge},
		{"huge password", ct, `{"login":"a","senha":"` + strings.Repeat(sentinelPassword, 4000) + `"}`, reasonTooLarge},
		{"empty body", ct, "", reasonEmpty},
		{"whitespace only", ct, "  \n", reasonEmpty},
		{"unknown field", ct, `{"login":"a","senha":"` + sentinelPassword + `","role_` + sentinelPassword + `":1}`, reasonMalformed},
		{"invalid json", ct, `{"login":"a","senha":"` + sentinelPassword + `"`, reasonMalformed},
		{"control byte in password", ct, `{"login":"a","senha":"` + sentinelPassword + "\x01" + `"}`, reasonMalformed},
		{"wrong type", ct, `{"login":"a","senha":["` + sentinelPassword + `"]}`, reasonMalformed},
		{"not an object", ct, `"` + sentinelPassword + `"`, reasonMalformed},
		{"second object", ct, valid + `{"login":"b"}`, reasonTrailing},
		{"trailing garbage", ct, valid + sentinelPassword, reasonTrailing},
		{"trailing data over the limit", ct, padTo(valid+"x", MaxBodyBytes+1), reasonTrailing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestSecurity(t, nil)
			var dst loginBody
			err := s.DecodeJSON(httptest.NewRecorder(), newJSONRequest(t, tt.contentType, tt.body), &dst)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("DecodeJSON() error = %v", err)
				}
				if dst.Login != "admin" || dst.Senha != sentinelPassword {
					t.Errorf("decoded %+v", dst)
				}
				return
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("DecodeJSON() error = %v, want ErrInvalidInput", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("DecodeJSON() error = %q, want reason %q", err, tt.wantErr)
			}
		})
	}
}

// TestDecodeJSONErrorsNeverQuoteTheBody guards the logs: an error from
// DecodeJSON may be logged, so it must be one of the fixed messages and
// never carry bytes or field names from the body.
func TestDecodeJSONErrorsNeverQuoteTheBody(t *testing.T) {
	t.Parallel()

	fixed := map[string]bool{}
	for _, reason := range []string{reasonContentType, reasonTooLarge, reasonEmpty, reasonMalformed, reasonTrailing} {
		fixed[invalidBody(reason).Error()] = true
	}

	bodies := []struct {
		contentType string
		body        string
	}{
		{"text/plain; charset=" + sentinelPassword, sentinelPassword},
		{"application/json", `{"senha":"` + sentinelPassword + "\x01" + `"}`},
		{"application/json", `{"senha":"` + sentinelPassword + `","` + sentinelPassword + `":1}`},
		{"application/json", `{"senha":` + sentinelPassword + `}`},
		{"application/json", `{"senha":{"` + sentinelPassword + `":1}}`},
		{"application/json", `{"senha":"x"}` + sentinelPassword},
		{"application/json", `{"senha":"` + strings.Repeat(sentinelPassword, 4000) + `"}`},
	}
	for _, b := range bodies {
		s := newTestSecurity(t, nil)
		var dst loginBody
		err := s.DecodeJSON(httptest.NewRecorder(), newJSONRequest(t, b.contentType, b.body), &dst)
		if err == nil {
			t.Fatalf("body %q was accepted", b.body)
		}
		msg := err.Error()
		if !fixed[msg] {
			t.Errorf("error %q is not one of the fixed messages", msg)
		}
		if strings.Contains(msg, sentinelPassword) || strings.Contains(msg, `\x01`) {
			t.Errorf("error %q quotes the body", msg)
		}
	}
}

func TestNormalizeLogin(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct{ in, want string }{
		{"admin", "admin"},
		{"Admin.Root", "admin.root"},
		{"ADMIN_1-x", "admin_1-x"},
		{" Padded ", " padded "}, // lowercase only, no trimming (D-12)
		{"", ""},
	}
	for _, tt := range tests {
		if got := s.NormalizeLogin(tt.in); got != tt.want {
			t.Errorf("NormalizeLogin(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateLogin(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		login string
		valid bool
	}{
		{"abc", true},
		{"admin", true},
		{"joao.silva_2-x", true},
		{strings.Repeat("a", 64), true},
		{"ab", false},
		{strings.Repeat("a", 65), false},
		{"", false},
		{"Admin", false}, // callers normalize first
		{"jo\u00e3o", false},
		{"a b c", false},
		{"admin\n", false},
		{"admin@x", false},
		{"../etc", false},
	}
	for _, tt := range tests {
		err := s.ValidateLogin(tt.login)
		if tt.valid && err != nil {
			t.Errorf("ValidateLogin(%q) = %v, want nil", tt.login, err)
		}
		if !tt.valid && !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("ValidateLogin(%q) = %v, want ErrInvalidInput", tt.login, err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"minimum", strings.Repeat("a", 15), true},
		{"maximum", strings.Repeat("a", 128), true},
		{"too short", strings.Repeat("a", 14), false},
		{"too long", strings.Repeat("a", 129), false},
		{"empty", "", false},
		// 15 code points but 30 bytes: length counts code points.
		{"multi-byte minimum", strings.Repeat("\u00e9", 15), true},
		{"multi-byte too long", strings.Repeat("\u00e9", 129), false},
		// "e" + combining acute (2 code points) composes to "\u00e9" (1) under NFKC.
		{"composes below minimum", strings.Repeat("e\u0301", 14), false},
		{"composes to the maximum", strings.Repeat("e\u0301", 128), true},
		// The ligature "\ufb03" (1 code point) expands to "ffi" (3) under NFKC.
		{"expands to the minimum", strings.Repeat("\ufb03", 5), true},
		{"expands above maximum", strings.Repeat("\ufb03", 43), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := s.ValidatePassword(tt.password)
			if tt.valid && err != nil {
				t.Errorf("ValidatePassword() = %v, want nil", err)
			}
			if !tt.valid && !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("ValidatePassword() = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestNormalizePasswordIsNFKC(t *testing.T) {
	t.Parallel()

	tests := []struct{ in, want string }{
		{"e\u0301", "\u00e9"},  // composition
		{"\uff21\uff11", "A1"}, // full-width letters and digits
		{"\ufb03", "ffi"},      // ligature
		{"plain ascii", "plain ascii"},
	}
	for _, tt := range tests {
		if got := normalizePassword(tt.in); got != tt.want {
			t.Errorf("normalizePassword(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseRole(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		raw     string
		want    domain.Role
		wantErr bool
	}{
		{"admin", domain.RoleAdmin, false},
		{"comum", domain.RoleComum, false},
		{"", "", true},
		{"Admin", "", true},
		{"ADMIN", "", true},
		{"Comum", "", true},
		{" admin", "", true},
		{"admin ", "", true},
		{"root", "", true},
		{"comum\x00", "", true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.raw), func(t *testing.T) {
			t.Parallel()
			got, err := s.ParseRole(tt.raw)
			if !tt.wantErr {
				if err != nil || got != tt.want {
					t.Fatalf("ParseRole(%q) = %q, %v; want %q", tt.raw, got, err, tt.want)
				}
				return
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("ParseRole(%q) error = %v, want ErrInvalidInput", tt.raw, err)
			}
			if got != "" {
				t.Errorf("ParseRole(%q) role = %q, want empty", tt.raw, got)
			}
			if tt.raw != "" && strings.Contains(err.Error(), tt.raw) {
				t.Errorf("error %q quotes the input", err)
			}
		})
	}
}
