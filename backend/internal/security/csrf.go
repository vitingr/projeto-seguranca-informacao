package security

import (
	"crypto/subtle"
	"net/http"
)

const CSRFHeaderName = "X-CSRF-Token"

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
