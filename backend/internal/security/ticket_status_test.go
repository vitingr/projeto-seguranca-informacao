package security

import (
	"errors"
	"io"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

var allStatuses = []domain.TicketStatus{
	domain.TicketStatusOpen, domain.TicketStatusInProgress,
	domain.TicketStatusResolved, domain.TicketStatusClosed,
}

func TestCheckStatusChange(t *testing.T) {
	t.Parallel()

	// BR-3, written out pair by pair so the test does not share the
	// implementation's table.
	allowed := map[[2]domain.TicketStatus]bool{
		{"aberto", "em_andamento"}:    true,
		{"aberto", "fechado"}:         true,
		{"em_andamento", "resolvido"}: true,
		{"em_andamento", "fechado"}:   true,
		{"resolvido", "em_andamento"}: true,
		{"resolvido", "fechado"}:      true,
	}
	s := newTestSecurity(t, io.Discard)
	pairs, accepted := 0, 0
	for _, from := range allStatuses {
		for _, to := range allStatuses {
			pairs++
			want := allowed[[2]domain.TicketStatus{from, to}]
			if want {
				accepted++
			}
			t.Run(string(from)+" → "+string(to), func(t *testing.T) {
				t.Parallel()
				err := s.CheckStatusChange(from, to)
				if want {
					if err != nil {
						t.Errorf("CheckStatusChange = %v, want allowed", err)
					}
					return
				}
				if from == domain.TicketStatusClosed {
					// "fechado" is final (BR-5): a conflict with the
					// ticket's state, 409.
					if !errors.Is(err, domain.ErrStaleStatus) || !errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrInvalidInput) {
						t.Errorf("CheckStatusChange = %v, want only ErrStaleStatus (409)", err)
					}
					return
				}
				// A pair BR-3 never allows is an invalid request, 422.
				if !errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrConflict) {
					t.Errorf("CheckStatusChange = %v, want only ErrInvalidInput (422)", err)
				}
				if !strings.Contains(err.Error(), reasonTransition) {
					t.Errorf("CheckStatusChange = %v, want reason %q", err, reasonTransition)
				}
			})
		}
	}
	if pairs != 16 || accepted != 6 {
		t.Fatalf("%d pairs, %d allowed; want 16 and 6", pairs, accepted)
	}

	for _, tt := range []struct{ from, to domain.TicketStatus }{
		{"", ""},
		{"", "aberto"},
		{"aberto", ""},
		{"cancelado", "fechado"},
		{"aberto", "cancelado"},
		{"Aberto", "em_andamento"},
		{"Fechado", "aberto"},
	} {
		if err := s.CheckStatusChange(tt.from, tt.to); !errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrConflict) {
			t.Errorf("CheckStatusChange(%q, %q) = %v, want ErrInvalidInput", tt.from, tt.to, err)
		}
	}
}

func TestAllowedNextStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from domain.TicketStatus
		want []domain.TicketStatus
	}{
		{domain.TicketStatusOpen, []domain.TicketStatus{domain.TicketStatusInProgress, domain.TicketStatusClosed}},
		{domain.TicketStatusInProgress, []domain.TicketStatus{domain.TicketStatusResolved, domain.TicketStatusClosed}},
		{domain.TicketStatusResolved, []domain.TicketStatus{domain.TicketStatusInProgress, domain.TicketStatusClosed}},
		{domain.TicketStatusClosed, nil},
		{"cancelado", nil},
		{"", nil},
	}
	s := newTestSecurity(t, io.Discard)
	for _, tt := range tests {
		got := AllowedNextStatuses(tt.from)
		if !slices.Equal(got, tt.want) {
			t.Errorf("AllowedNextStatuses(%q) = %v, want %v", tt.from, got, tt.want)
		}
		// The list and the rule agree.
		for _, to := range got {
			if err := s.CheckStatusChange(tt.from, to); err != nil {
				t.Errorf("AllowedNextStatuses(%q) lists %q, but CheckStatusChange = %v", tt.from, to, err)
			}
		}
	}

	// A copy: changing it does not change the rule.
	got := AllowedNextStatuses(domain.TicketStatusOpen)
	got[0] = domain.TicketStatusResolved
	if err := s.CheckStatusChange(domain.TicketStatusOpen, domain.TicketStatusResolved); err == nil {
		t.Error("changing the returned slice changed the transitions")
	}
}

func TestParseStatus(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	for _, status := range allStatuses {
		got, err := s.ParseStatus(string(status))
		if err != nil || got != status {
			t.Errorf("ParseStatus(%q) = %q, %v; want %q, nil", status, got, err, status)
		}
	}
	for _, raw := range []string{"", "cancelado", "Aberto", "FECHADO", " aberto", "aberto ", "em andamento", "Em andamento"} {
		got, err := s.ParseStatus(raw)
		if !errors.Is(err, domain.ErrInvalidInput) || got != "" {
			t.Errorf("ParseStatus(%q) = %q, %v; want ErrInvalidInput", raw, got, err)
		}
	}
}

func TestDecodeStatusChange(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	decode := func(t *testing.T, contentType, body string) (string, string, error) {
		t.Helper()
		return s.DecodeStatusChange(httptest.NewRecorder(), newJSONRequest(t, contentType, body))
	}

	accepted := []struct {
		name, body, from, to string
	}{
		{"both keys", `{"de":"aberto","para":"em_andamento"}`, "aberto", "em_andamento"},
		{"other order", `{"para":"fechado","de":"resolvido"}`, "resolvido", "fechado"},
		{"unknown codes decode (ParseStatus refuses them)", `{"de":"x","para":""}`, "x", ""},
	}
	for _, tt := range accepted {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			from, to, err := decode(t, "application/json", tt.body)
			if err != nil || from != tt.from || to != tt.to {
				t.Errorf("DecodeStatusChange(%q) = %q, %q, %v; want %q, %q, nil", tt.body, from, to, err, tt.from, tt.to)
			}
		})
	}

	const ct = "application/json"
	refused := []struct {
		name, contentType, body, reason string
	}{
		// Missing keys.
		{"empty object", ct, `{}`, reasonNoFrom},
		{"no de", ct, `{"para":"fechado"}`, reasonNoFrom},
		{"no para", ct, `{"de":"aberto"}`, reasonNoTo},
		{"de null", ct, `{"de":null,"para":"fechado"}`, reasonNoFrom},
		{"para null", ct, `{"de":"aberto","para":null}`, reasonNoTo},

		// Server-owned and unknown keys (S-8).
		{"actor", ct, `{"de":"aberto","para":"fechado","actor":1}`, reasonMalformed},
		{"updated_at", ct, `{"de":"aberto","para":"fechado","updated_at":"2020-01-01T00:00:00Z"}`, reasonMalformed},
		{"id", ct, `{"de":"aberto","para":"fechado","id":1}`, reasonMalformed},
		{"status", ct, `{"status":"fechado"}`, reasonMalformed},
		{"keys in english", ct, `{"from":"aberto","to":"fechado"}`, reasonMalformed},

		// Key spelling.
		{"De", ct, `{"De":"aberto","para":"fechado"}`, reasonMalformed},
		{"PARA", ct, `{"de":"aberto","PARA":"fechado"}`, reasonMalformed},
		{"duplicate de", ct, `{"de":"aberto","de":"resolvido","para":"fechado"}`, reasonDuplicateKey},
		{"duplicate para in another case", ct, `{"de":"aberto","para":"fechado","Para":"resolvido"}`, reasonDuplicateKey},

		// Wrong JSON types.
		{"de number", ct, `{"de":1,"para":"fechado"}`, reasonMalformed},
		{"para bool", ct, `{"de":"aberto","para":true}`, reasonMalformed},
		{"para array", ct, `{"de":"aberto","para":["fechado"]}`, reasonMalformed},
		{"body array", ct, `[{"de":"aberto","para":"fechado"}]`, reasonMalformed},
		{"body null", ct, `null`, reasonMalformed},

		// The envelope rules of DecodeJSON.
		{"trailing data", ct, `{"de":"aberto","para":"fechado"}{}`, reasonTrailing},
		{"wrong content type", "text/plain", `{"de":"aberto","para":"fechado"}`, reasonContentType},
		{"empty body", ct, ``, reasonEmpty},
	}
	for _, tt := range refused {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			from, to, err := decode(t, tt.contentType, tt.body)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput (400)", err)
			}
			if err.Error() != invalidBody(tt.reason).Error() {
				t.Errorf("error = %q, want the fixed %q", err, invalidBody(tt.reason))
			}
			if from != "" || to != "" {
				t.Errorf("values %q, %q on error, want empty", from, to)
			}
		})
	}
}
