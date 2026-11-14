package web

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"time"
)

type requestIDKey struct{}

// RequestIDFrom retorna o request ID do contexto ou "" se ausente.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// requestID gera um identificador aleatório por requisição, coloca-o no
// contexto e no cabeçalho X-Request-Id. O valor enviado pelo cliente é
// ignorado para impedir injeção em logs.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := rand.Text()
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// accessLog registra uma linha por requisição. Registra apenas o caminho,
// nunca a query string nem o corpo (podem conter dados sensíveis).
func accessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.InfoContext(r.Context(), "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			)
		})
	}
}

// statusRecorder captura o status escrito pelo handler.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	s.wroteHeader = true
	return s.ResponseWriter.Write(b)
}

// Unwrap permite que http.ResponseController alcance o writer original.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

// NewLogHandler envolve h para incluir automaticamente o request_id do
// contexto em todo registro feito com os métodos *Context do slog.
func NewLogHandler(h slog.Handler) slog.Handler {
	return logHandler{Handler: h}
}

type logHandler struct {
	slog.Handler
}

func (h logHandler) Handle(ctx context.Context, rec slog.Record) error {
	if id := RequestIDFrom(ctx); id != "" {
		rec.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, rec)
}

func (h logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return logHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h logHandler) WithGroup(name string) slog.Handler {
	return logHandler{Handler: h.Handler.WithGroup(name)}
}
