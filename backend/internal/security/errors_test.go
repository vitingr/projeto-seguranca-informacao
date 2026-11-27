package security

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

func wrapTwice(err error) error {
	return fmt.Errorf("service: op: %w", fmt.Errorf("repository: op: %w", err))
}

func TestRespondErrorMapsDomainErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantLevel  string
	}{
		{"invalid input", wrapTwice(domain.ErrInvalidInput), http.StatusBadRequest, "WARN"},
		{"not found", wrapTwice(domain.ErrNotFound), http.StatusNotFound, "WARN"},
		{"conflict", wrapTwice(domain.ErrConflict), http.StatusConflict, "WARN"},
		{"invalid credentials", wrapTwice(domain.ErrInvalidCredentials), http.StatusInternalServerError, "ERROR"},
		{"deadline exceeded", wrapTwice(context.DeadlineExceeded), http.StatusInternalServerError, "ERROR"},
		{"plain error with a domain text", errors.New("conflict"), http.StatusInternalServerError, "ERROR"},
		{"nil", nil, http.StatusInternalServerError, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			s := newTestSecurity(t, &logs)
			rr := httptest.NewRecorder()
			s.RespondError(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/cadastro", http.NoBody), tt.err)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.err != nil && strings.Contains(rr.Body.String(), tt.err.Error()) {
				t.Errorf("body %q has the error text", rr.Body.String())
			}
			var entry struct {
				Level  string `json:"level"`
				Error  string `json:"error"`
				Status int    `json:"status"`
			}
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("log %q: %v", logs.String(), err)
			}
			if entry.Level != tt.wantLevel || entry.Status != tt.wantStatus {
				t.Errorf("log level %q, status %d; want %q, %d", entry.Level, entry.Status, tt.wantLevel, tt.wantStatus)
			}
			if tt.err != nil && entry.Error != tt.err.Error() {
				t.Errorf("logged error = %q, want %q", entry.Error, tt.err.Error())
			}
		})
	}
}

func TestRespondErrorNeverLogsTheBody(t *testing.T) {
	t.Parallel()

	bodies := []struct {
		contentType string
		body        string
	}{
		{"text/plain", `{"senha":"` + sentinelPassword + `"}`},
		{"application/json", `{"senha":` + sentinelPassword + `}`},
		{"application/json", `{"senha":"` + sentinelPassword + `","` + sentinelPassword + `":1}`},
		{"application/json", `{"senha":"x"}` + sentinelPassword},
	}
	for _, b := range bodies {
		var logs bytes.Buffer
		s := newTestSecurity(t, &logs)
		var dst loginBody
		req := newJSONRequest(t, b.contentType, b.body)
		err := s.DecodeJSON(httptest.NewRecorder(), req, &dst)
		if err == nil {
			t.Fatalf("body %q was accepted", b.body)
		}
		rr := httptest.NewRecorder()
		s.RespondError(rr, req, err)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", b.body, rr.Code)
		}
		if strings.Contains(logs.String(), sentinelPassword) {
			t.Errorf("the log has the password:\n%s", logs.String())
		}
	}
}
