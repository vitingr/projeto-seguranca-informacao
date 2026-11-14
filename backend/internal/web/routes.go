package web

import (
	"net/http"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

// route declara uma rota: método, caminho, handler e política de acesso.
// Toda rota DEVE declarar a política; o valor zero de security.Policy
// nega tudo (garantido também por teste).
type route struct {
	method  string // vazio = qualquer método (apenas para o catch-all)
	path    string
	handler http.Handler
	policy  security.Policy
}

// pattern retorna o padrão do http.ServeMux (ex.: "POST /api/login").
func (rt route) pattern() string {
	if rt.method == "" {
		return rt.path
	}
	return rt.method + " " + rt.path
}

// routes é a tabela declarativa de rotas da API.
//
// As páginas /login, /cadastro, /administradores e /usuarios agora vivem no
// frontend; cada uma tem aqui o endpoint correspondente, com a mesma
// política de acesso. A proteção de rotas no frontend é só experiência do
// usuário: quem decide o acesso é esta tabela.
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
		// Catch-all: qualquer outro caminho/método recebe 404 em JSON.
		{"", "/", http.HandlerFunc(h.notFound), public},
	}
}
