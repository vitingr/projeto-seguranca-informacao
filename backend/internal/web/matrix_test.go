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

// matrixPostBody is a valid post, and matrixPostPath names the post that
// each matrix test seeds for the comum user, so comum gets 2xx on every
// post route (005).
const (
	matrixPostBody = `{"title":"Título","body":"Corpo do post."}`
	matrixPostPath = "/api/posts/1"
)

// matrixRemovalPath and matrixRemovalBody remove the seeded post, and
// matrixAuditPath lists the audit log (006).
const (
	matrixRemovalPath = "/api/posts/1/removal"
	matrixRemovalBody = `{"reason":"Conteúdo impróprio."}`
	matrixAuditPath   = "/api/audit-events"
)

// TestAccessMatrix covers the access matrix (003 D-1, D-2, the four post
// routes of 005 and the two routes of 006) through the full NewHandler
// chain. Every caller holds
// a session and sends a valid CSRF token on mutations, so a 401 or 403 can
// only come from Authorize; the log reason proves it.
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
		{anonymous, http.MethodGet, "/api/posts", http.StatusUnauthorized},
		{anonymous, http.MethodPost, "/api/posts", http.StatusUnauthorized},
		{anonymous, http.MethodPut, matrixPostPath, http.StatusUnauthorized},
		{anonymous, http.MethodDelete, matrixPostPath, http.StatusUnauthorized},
		{anonymous, http.MethodPost, matrixRemovalPath, http.StatusUnauthorized},
		{anonymous, http.MethodGet, matrixAuditPath, http.StatusUnauthorized},

		{comumLogin, http.MethodGet, "/api/administradores", http.StatusForbidden},
		{comumLogin, http.MethodGet, "/api/usuarios", http.StatusOK},
		{comumLogin, http.MethodPost, "/api/cadastro", http.StatusForbidden},
		{comumLogin, http.MethodGet, "/api/me", http.StatusOK},
		{comumLogin, http.MethodPost, "/api/logout", http.StatusNoContent},
		{comumLogin, http.MethodGet, "/api/posts", http.StatusOK},
		{comumLogin, http.MethodPost, "/api/posts", http.StatusCreated},
		{comumLogin, http.MethodPut, matrixPostPath, http.StatusOK},
		{comumLogin, http.MethodDelete, matrixPostPath, http.StatusNoContent},
		{comumLogin, http.MethodPost, matrixRemovalPath, http.StatusForbidden}, // only admins moderate (006 S-3)
		{comumLogin, http.MethodGet, matrixAuditPath, http.StatusForbidden},

		{flowLogin, http.MethodGet, "/api/administradores", http.StatusOK},
		{flowLogin, http.MethodGet, "/api/usuarios", http.StatusForbidden}, // admins do not reach /usuarios
		{flowLogin, http.MethodPost, "/api/cadastro", http.StatusCreated},
		{flowLogin, http.MethodGet, "/api/me", http.StatusOK},
		{flowLogin, http.MethodPost, "/api/logout", http.StatusNoContent},
		{flowLogin, http.MethodGet, "/api/posts", http.StatusOK}, // admins read the wall to moderate it (006 D-1)
		{flowLogin, http.MethodPost, "/api/posts", http.StatusForbidden},
		{flowLogin, http.MethodPut, matrixPostPath, http.StatusForbidden},
		{flowLogin, http.MethodDelete, matrixPostPath, http.StatusForbidden},
		{flowLogin, http.MethodPost, matrixRemovalPath, http.StatusNoContent},
		{flowLogin, http.MethodGet, matrixAuditPath, http.StatusOK},
	}
	if len(tests) != 33 {
		t.Fatalf("matrix has %d cells, want 33 (003 D-2, 4 post routes and 2 routes of 006, × 3 callers)", len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.caller+" "+tt.method+" "+tt.path, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			f := newFlowAppWithLog(t, &logBuf)
			f.posts.seed(t, 2, "do comum") // post 1, written by comumLogin
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
			if tt.method != http.MethodGet {
				headers[security.CSRFHeaderName] = token
			}
			switch {
			case tt.path == "/api/cadastro":
				body = matrixCadastroBody
			case tt.method == http.MethodPost && tt.path == "/api/posts", tt.method == http.MethodPut:
				body = matrixPostBody
			case tt.path == matrixRemovalPath:
				body = matrixRemovalBody
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
