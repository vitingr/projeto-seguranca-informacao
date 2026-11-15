package security

import (
	"crypto/subtle"
	"net/http"
)

// CSRFHeaderName é o cabeçalho em que o frontend envia o token CSRF em toda
// requisição que altera estado. O token é obtido em GET /api/csrf.
const CSRFHeaderName = "X-CSRF-Token"

// CSRF aplica o padrão synchronizer token adaptado a uma API consumida por
// fetch: em métodos que alteram estado (POST, PUT, PATCH, DELETE...), o
// cabeçalho CSRFHeaderName deve ser igual ao token da sessão (comparação em
// tempo constante). Métodos seguros (GET, HEAD, OPTIONS) passam direto e não
// devem alterar estado.
//
// Two checks run on unsafe methods, both answering 403 (D-5):
//  1. Fetch Metadata (http.CrossOriginProtection): Sec-Fetch-Site must be
//     same-origin or none; without it, an Origin whose host differs from
//     Host is rejected; without both (Bruno, curl) the request goes on.
//  2. The token: the request needs a session (from GET /api/csrf) and
//     X-CSRF-Token equal to its token.
func (s *Security) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if err := s.crossOrigin.Check(r); err != nil {
			s.deny(w, r, http.StatusForbidden, "csrf: cross-origin request")
			return
		}
		st, ok := sessionFrom(r.Context())
		if !ok || st.csrfToken == "" {
			s.deny(w, r, http.StatusForbidden, "csrf: no session")
			return
		}
		got := r.Header.Get(CSRFHeaderName)
		if subtle.ConstantTimeCompare([]byte(got), []byte(st.csrfToken)) != 1 {
			s.deny(w, r, http.StatusForbidden, "csrf: token mismatch")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
