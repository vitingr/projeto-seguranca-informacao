package security

import "net/http"

// contentSecurityPolicy da API: as respostas são JSON e nunca devem ser
// interpretadas como documento nem embutidas em frames. A CSP das páginas é
// responsabilidade do frontend.
const contentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'"

// SecurityHeaders define cabeçalhos de segurança em todas as respostas,
// antes de qualquer handler escrever o corpo.
func (s *Security) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		// Respostas com dados de usuário não devem ficar em cache.
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
