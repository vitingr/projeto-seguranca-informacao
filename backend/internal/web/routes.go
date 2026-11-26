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
		{"", "/", http.HandlerFunc(h.notFound), public},
	}
}
