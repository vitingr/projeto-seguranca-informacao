package web

import (
	"net/http"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

type route struct {
	method  string
	path    string
	handler http.Handler
	policy  security.Policy
}

func (rt route) pattern() string {
	if rt.method == "" {
		return rt.path
	}
	return rt.method + " " + rt.path
}

// routeQuery lists, by pattern, the query parameters a route reads (spec
// 003 section 4). A route that is not here reads none. The handlers refuse
// any other parameter with 400, and the contract test compares this table
// with the parameters of api/openapi.json.
var routeQuery = map[string][]string{
	"GET /api/tickets":                {"status", "tipo", "pagina"},
	"GET /api/tickets/{id}/mensagens": {"antes"},
}

func (h *handlers) routes() []route {
	admin := security.RequireRole(domain.RoleAdmin)
	comum := security.RequireRole(domain.RoleComum)
	autenticado := security.RequireRole(domain.RoleAdmin, domain.RoleComum)
	public := security.Public()

	return []route{
		{http.MethodGet, "/healthz", http.HandlerFunc(h.healthz), public},
		{http.MethodGet, "/api/csrf", http.HandlerFunc(h.csrf), public},
		{http.MethodPost, "/api/login", http.HandlerFunc(h.login), public},
		{http.MethodPost, "/api/logout", http.HandlerFunc(h.logout), autenticado},
		{http.MethodGet, "/api/me", http.HandlerFunc(h.me), autenticado},
		{http.MethodPost, "/api/cadastro", http.HandlerFunc(h.cadastro), admin},
		{http.MethodGet, "/api/administradores", http.HandlerFunc(h.administradores), admin},
		{http.MethodGet, "/api/usuarios", http.HandlerFunc(h.usuarios), comum},
		{http.MethodPost, "/api/tickets", chain(http.HandlerFunc(h.createTicket), h.sec.RateLimit(h.ticketLimiter)), comum},
		{http.MethodGet, "/api/tickets", http.HandlerFunc(h.listOwnTickets), comum},
		{http.MethodGet, "/api/tickets/{id}", chain(http.HandlerFunc(h.getOwnTicket), h.sec.TicketAccess), comum},
		{http.MethodGet, "/api/tickets/{id}/mensagens", chain(http.HandlerFunc(h.getOwnThread), h.sec.TicketAccess), comum},
		{"", "/", http.HandlerFunc(h.notFound), public},
	}
}
