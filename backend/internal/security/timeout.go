package security

import (
	"context"
	"net/http"
)

// RequestTimeout gives every request a context deadline of
// Options.RequestTimeout (D-21).
//
// The http.Server timeouts do not cancel the request context on HTTP/1.1,
// so without this deadline a request could wait for a hash slot (D-19), or
// for the database, for as long as the client keeps the connection open.
// When the deadline passes, the waiting call returns the context error and
// the handler answers the generic 500.
func (s *Security) RequestTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), s.reqTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
