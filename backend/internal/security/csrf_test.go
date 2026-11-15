package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// csrfFixture is a Security with one anonymous session, as GET /api/csrf
// creates it.
type csrfFixture struct {
	s     *Security
	token string // session cookie
	csrf  string
}

func newCSRFFixture(t *testing.T) csrfFixture {
	t.Helper()
	s := newSessionSecurity(t, newMemStore(), newClock(), nil)
	rr := httptest.NewRecorder()
	csrf, err := s.CSRFToken(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/csrf", http.NoBody))
	if err != nil {
		t.Fatal(err)
	}
	return csrfFixture{s: s, token: sessionCookieFrom(t, rr).Value, csrf: csrf}
}

// do sends a request through Session and CSRF and reports whether the
// handler ran and the status.
func (f csrfFixture) do(t *testing.T, method string, withCookie bool, headers map[string]string) (bool, int) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, "https://api.example/api/login", http.NoBody)
	if withCookie {
		req.AddCookie(reqCookie(f.token))
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	var called bool
	rr := httptest.NewRecorder()
	f.s.Session(f.s.CSRF(okHandler(&called))).ServeHTTP(rr, req)
	return called, rr.Code
}

func TestCSRFSafeMethodsPass(t *testing.T) {
	t.Parallel()

	f := newCSRFFixture(t)
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		// Even cross-site and without a session: safe methods must not
		// change state.
		called, _ := f.do(t, method, false, map[string]string{"Sec-Fetch-Site": "cross-site"})
		if !called {
			t.Errorf("%s was blocked", method)
		}
	}
}

func TestCSRFTokenCheck(t *testing.T) {
	t.Parallel()

	f := newCSRFFixture(t)
	tests := []struct {
		name       string
		method     string
		withCookie bool
		token      string // "" = header absent; "<valid>" = the session token
		wantCalled bool
	}{
		{"valid token", http.MethodPost, true, "<valid>", true},
		{"valid token, PUT", http.MethodPut, true, "<valid>", true},
		{"valid token, DELETE", http.MethodDelete, true, "<valid>", true},
		{"no header", http.MethodPost, true, "", false},
		{"wrong token", http.MethodPost, true, "AAAAAAAAAAAAAAAAAAAAAAAAAA", false},
		{"token with a suffix", http.MethodPost, true, "<valid>x", false},
		{"token prefix", http.MethodPost, true, "<prefix>", false},
		{"lowercase token", http.MethodPost, true, "<lower>", false},
		{"no session", http.MethodPost, false, "<valid>", false},
		{"unknown method", "PROPFIND", true, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			headers := map[string]string{}
			switch tt.token {
			case "":
			case "<valid>":
				headers[CSRFHeaderName] = f.csrf
			case "<valid>x":
				headers[CSRFHeaderName] = f.csrf + "x"
			case "<prefix>":
				headers[CSRFHeaderName] = f.csrf[:10]
			case "<lower>":
				headers[CSRFHeaderName] = strings.ToLower(f.csrf)
			default:
				headers[CSRFHeaderName] = tt.token
			}
			called, code := f.do(t, tt.method, tt.withCookie, headers)
			if called != tt.wantCalled {
				t.Errorf("handler ran = %v, want %v", called, tt.wantCalled)
			}
			if !tt.wantCalled && code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", code)
			}
		})
	}
}

func TestCSRFFetchMetadata(t *testing.T) {
	t.Parallel()

	f := newCSRFFixture(t)
	tests := []struct {
		name       string
		headers    map[string]string
		wantCalled bool
	}{
		{"no Sec-Fetch-Site, no Origin (Bruno, curl)", nil, true},
		{"same-origin", map[string]string{"Sec-Fetch-Site": "same-origin"}, true},
		{"none (typed URL, bookmark)", map[string]string{"Sec-Fetch-Site": "none"}, true},
		{"cross-site", map[string]string{"Sec-Fetch-Site": "cross-site"}, false},
		{"same-site (another subdomain)", map[string]string{"Sec-Fetch-Site": "same-site"}, false},
		{"cross-site with a matching Origin", map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://api.example"}, false},
		{"Origin matching Host", map[string]string{"Origin": "https://api.example"}, true},
		{"Origin of another host", map[string]string{"Origin": "https://evil.example"}, false},
		{"Origin of another port", map[string]string{"Origin": "https://api.example:8443"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			headers := map[string]string{CSRFHeaderName: f.csrf} // a valid token in every case
			for k, v := range tt.headers {
				headers[k] = v
			}
			called, code := f.do(t, http.MethodPost, true, headers)
			if called != tt.wantCalled {
				t.Errorf("handler ran = %v, want %v", called, tt.wantCalled)
			}
			if !tt.wantCalled && code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", code)
			}
		})
	}
}
