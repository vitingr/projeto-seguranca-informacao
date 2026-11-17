package security

import (
	"context"
	"net/http"
)

func (s *Security) RequestTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), s.reqTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
