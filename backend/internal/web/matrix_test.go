package web

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

const matrixCadastroBody = `{"login":"novo.usuario","senha":"a long enough password","papel":"comum"}`

const matrixTicket = "/api/tickets/1"

const matrixAdminTicket = "/api/administradores/tickets/1"

const matrixReplyBody = `{"texto":"Alguma novidade?"}`

const matrixTicketBody = `{"tipo":"outro","dados":{"titulo":"Dúvida","descricao":"Como peço acesso à VPN?"}}`

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
		{flowLogin, http.MethodGet, "/api/usuarios", http.StatusForbidden},
		{flowLogin, http.MethodPost, "/api/cadastro", http.StatusCreated},
		{flowLogin, http.MethodGet, "/api/me", http.StatusOK},
		{flowLogin, http.MethodPost, "/api/logout", http.StatusNoContent},
		{flowLogin, http.MethodPost, "/api/tickets", http.StatusForbidden},

		{anonymous, http.MethodGet, "/api/tickets", http.StatusUnauthorized},
		{comumLogin, http.MethodGet, "/api/tickets", http.StatusOK},
		{flowLogin, http.MethodGet, "/api/tickets", http.StatusForbidden},
		{anonymous, http.MethodGet, matrixTicket, http.StatusUnauthorized},
		{comumLogin, http.MethodGet, matrixTicket, http.StatusOK},
		{flowLogin, http.MethodGet, matrixTicket, http.StatusForbidden},
		{anonymous, http.MethodGet, matrixTicket + "/mensagens", http.StatusUnauthorized},
		{comumLogin, http.MethodGet, matrixTicket + "/mensagens", http.StatusOK},
		{flowLogin, http.MethodGet, matrixTicket + "/mensagens", http.StatusForbidden},
		{anonymous, http.MethodPost, matrixTicket + "/mensagens", http.StatusUnauthorized},
		{comumLogin, http.MethodPost, matrixTicket + "/mensagens", http.StatusCreated},
		{flowLogin, http.MethodPost, matrixTicket + "/mensagens", http.StatusForbidden},
		{anonymous, http.MethodGet, "/api/administradores/tickets", http.StatusUnauthorized},
		{comumLogin, http.MethodGet, "/api/administradores/tickets", http.StatusForbidden},
		{flowLogin, http.MethodGet, "/api/administradores/tickets", http.StatusOK},
		{anonymous, http.MethodGet, "/api/administradores/tickets/autores", http.StatusUnauthorized},
		{comumLogin, http.MethodGet, "/api/administradores/tickets/autores", http.StatusForbidden},
		{flowLogin, http.MethodGet, "/api/administradores/tickets/autores", http.StatusOK},
		{anonymous, http.MethodGet, matrixAdminTicket, http.StatusUnauthorized},
		{comumLogin, http.MethodGet, matrixAdminTicket, http.StatusForbidden},
		{flowLogin, http.MethodGet, matrixAdminTicket, http.StatusOK},
		{anonymous, http.MethodGet, matrixAdminTicket + "/mensagens", http.StatusUnauthorized},
		{comumLogin, http.MethodGet, matrixAdminTicket + "/mensagens", http.StatusForbidden},
		{flowLogin, http.MethodGet, matrixAdminTicket + "/mensagens", http.StatusOK},
		{anonymous, http.MethodPost, matrixAdminTicket + "/mensagens", http.StatusUnauthorized},
		{comumLogin, http.MethodPost, matrixAdminTicket + "/mensagens", http.StatusForbidden},
		{flowLogin, http.MethodPost, matrixAdminTicket + "/mensagens", http.StatusCreated},
		{anonymous, http.MethodPatch, matrixAdminTicket + "/status", http.StatusUnauthorized},
		{comumLogin, http.MethodPatch, matrixAdminTicket + "/status", http.StatusForbidden},
		{flowLogin, http.MethodPatch, matrixAdminTicket + "/status", http.StatusOK},
	}
	if len(tests) != 48 {
		t.Fatalf("matrix has %d cells, want 48", len(tests))
	}
	for _, tt := range tests {
		t.Run(tt.caller+" "+tt.method+" "+tt.path, func(t *testing.T) {
			t.Parallel()

			var logBuf bytes.Buffer
			tracking := newTrackingApp(t, &logBuf)
			tracking.store.seed(trackMariaID, domain.TicketTypeOther, "Dúvida sobre a VPN")
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
			if tt.method != http.MethodGet {
				headers[security.CSRFHeaderName] = token
				switch tt.path {
				case "/api/cadastro":
					body = matrixCadastroBody
				case "/api/tickets":
					body = matrixTicketBody
				case matrixTicket + "/mensagens", matrixAdminTicket + "/mensagens":
					body = matrixReplyBody
				case matrixAdminTicket + "/status":
					body = `{"de":"aberto","para":"em_andamento"}`
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
