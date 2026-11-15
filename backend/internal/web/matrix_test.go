package web

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"backend_si/internal/security"
)

// matrixCadastroBody is a well-formed registration body, so a status other
// than 201 comes from the middleware chain, not from validation.
const matrixCadastroBody = `{"login":"novo.usuario","senha":"a long enough password","papel":"comum"}`

// TestAccessMatrix covers the 15-cell access matrix (003 D-1, D-2) through
// the full NewHandler chain. Every caller holds a session and sends a valid
// CSRF token on POSTs, so a 401 or 403 can only come from Authorize; the log
// reason proves it.
func TestAccessMatrix(t *testing.T) {
	t.Parallel()

	const anonymous = "anonymous"
	tests := []struct {
		caller     string
		method     string
		path       string
		wantStatus int
	}{
		{anonymous, http.MethodGet, "/api/administradores", http.StatusUnauthorized},
		{anonymous, http.MethodGet, "/api/usuarios", http.StatusUnauthorized},
		{anonymous, http.MethodPost, "/api/cadastro", http.StatusUnauthorized},
		{anonymous, http.MethodGet, "/api/me", http.StatusUnauthorized},
		{anonymous, http.MethodPost, "/api/logout", http.StatusUnauthorized},

		{comumLogin, http.MethodGet, "/api/administradores", http.StatusForbidden},
		{comumLogin, http.MethodGet, "/api/usuarios", http.StatusOK},
		{comumLogin, http.MethodPost, "/api/cadastro", http.StatusForbidden},
		{comumLogin, http.MethodGet, "/api/me", http.StatusOK},
		{comumLogin, http.MethodPost, "/api/logout", http.StatusNoContent},

		{flowLogin, http.MethodGet, "/api/administradores", http.StatusOK},
		{flowLogin, http.MethodGet, "/api/usuarios", http.StatusForbidden}, // admins do not reach /usuarios
		{flowLogin, http.MethodPost, "/api/cadastro", http.StatusCreated},
		{flowLogin, http.MethodGet, "/api/me", http.StatusOK},
		{flowLogin, http.MethodPost, "/api/logout", http.StatusNoContent},
	}
	if len(tests) != 15 {
		t.Fatalf("matrix has %d cells, want 15 (D-2)", len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.caller+" "+tt.method+" "+tt.path, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			f := newFlowAppWithLog(t, &logBuf)
			var (
				cookie *http.Cookie
				token  string
			)
			if tt.caller == anonymous {
				cookie, token = f.csrf(t, nil)
			} else {
				cookie, token = f.loginAs(t, tt.caller)
			}

			body := ""
			headers := map[string]string{}
			if tt.method == http.MethodPost {
				headers[security.CSRFHeaderName] = token
				if tt.path == "/api/cadastro" {
					body = matrixCadastroBody
				}
			}

			logBuf.Reset()
			rr := f.send(t, tt.method, tt.path, body, cookie, headers)
			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rr.Code, tt.wantStatus, rr.Body.String())
			}

			logged := logBuf.String()
			if strings.Contains(logged, `"reason":"csrf:`) {
				t.Errorf("CSRF denied the request, want it to reach Authorize:\n%s", logged)
			}
			wantReason := ""
			switch tt.wantStatus {
			case http.StatusUnauthorized:
				wantReason = `"reason":"authz: anonymous"`
			case http.StatusForbidden:
				wantReason = `"reason":"authz: role not allowed"`
			}
			if wantReason != "" && !strings.Contains(logged, wantReason) {
				t.Errorf("log has no %s:\n%s", wantReason, logged)
			}
			if wantReason == "" && strings.Contains(logged, `"reason":`) {
				t.Errorf("allowed request logged a denial:\n%s", logged)
			}
		})
	}
}
