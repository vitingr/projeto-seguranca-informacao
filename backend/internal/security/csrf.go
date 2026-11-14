package security

import "net/http"

// CSRFHeaderName é o cabeçalho em que o frontend envia o token CSRF em toda
// requisição que altera estado. O token é obtido em GET /api/csrf.
const CSRFHeaderName = "X-CSRF-Token"

// CSRF aplica o padrão synchronizer token adaptado a uma API consumida por
// fetch: em métodos que alteram estado (POST, PUT, PATCH, DELETE...), o
// cabeçalho CSRFHeaderName deve ser igual ao token da sessão (comparação em
// tempo constante). Métodos seguros (GET, HEAD, OPTIONS) passam direto e não
// devem alterar estado.
func (s *Security) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		// TODO(etapa 2):
		//   1. rejeitar se Sec-Fetch-Site existir e for diferente de
		//      "same-origin" (ou, na ausência dele, se Origin não for a
		//      origem esperada);
		//   2. obter o token da sessão e comparar com r.Header.Get(CSRFHeaderName)
		//      usando subtle.ConstantTimeCompare.
		//
		// Stub fail-closed: toda requisição que altera estado é rejeitada.
		s.deny(w, r, http.StatusForbidden, "csrf: validação não implementada")
	})
}

// CSRFToken retorna o token CSRF da sessão da requisição, entregue ao
// frontend por GET /api/csrf.
func CSRFToken(r *http.Request) string {
	// TODO(etapa 2): ler o token da sessão no contexto.
	return ""
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
