package web

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// matrixCadastroBody is a well-formed registration body, so a status other
// than 201 comes from the middleware chain, not from validation.
const matrixCadastroBody = `{"login":"novo.usuario","senha":"a long enough password","papel":"comum"}`

// matrixTicketBody is a valid ticket request, for the same reason.
const matrixTicketBody = `{"tipo":"outro","dados":{"titulo":"Dúvida","descricao":"Como peço acesso à VPN?"}}`

// TestAccessMatrix covers the access matrix (003 D-1, D-2: 15 cells; spec
// 001-ticket-builder S-6 adds POST /api/tickets: 18 cells) through
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
		{anonymous, http.MethodPost, "/api/tickets", http.StatusUnauthorized},

		{comumLogin, http.MethodGet, "/api/administradores", http.StatusForbidden},
		{comumLogin, http.MethodGet, "/api/usuarios", http.StatusOK},
		{comumLogin, http.MethodPost, "/api/cadastro", http.StatusForbidden},
		{comumLogin, http.MethodGet, "/api/me", http.StatusOK},
		{comumLogin, http.MethodPost, "/api/logout", http.StatusNoContent},
		{comumLogin, http.MethodPost, "/api/tickets", http.StatusCreated},

		{flowLogin, http.MethodGet, "/api/administradores", http.StatusOK},
		{flowLogin, http.MethodGet, "/api/usuarios", http.StatusForbidden}, // admins do not reach /usuarios
		{flowLogin, http.MethodPost, "/api/cadastro", http.StatusCreated},
		{flowLogin, http.MethodGet, "/api/me", http.StatusOK},
		{flowLogin, http.MethodPost, "/api/logout", http.StatusNoContent},
		{flowLogin, http.MethodPost, "/api/tickets", http.StatusForbidden}, // admins do not create tickets

		// Spec 003 (plan section 7): each ticket tracking route x the three
		// callers. The {id} routes use matrixTicket, a ticket of comumLogin.
		{anonymous, http.MethodGet, "/api/tickets", http.StatusUnauthorized},
		{comumLogin, http.MethodGet, "/api/tickets", http.StatusOK},
		{flowLogin, http.MethodGet, "/api/tickets", http.StatusForbidden},
	}
	if len(tests) != 21 {
		t.Fatalf("matrix has %d cells, want 21 (18 of spec 001, 3 per spec 003 route)", len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.caller+" "+tt.method+" "+tt.path, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			tracking := newTrackingApp(t, &logBuf)
			tracking.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida sobre a VPN") // matrixTicket
			f := tracking.flowApp
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
				switch tt.path {
				case "/api/cadastro":
					body = matrixCadastroBody
				case "/api/tickets":
					body = matrixTicketBody
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
