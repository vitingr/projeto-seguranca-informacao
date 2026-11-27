package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"backend_si/internal/domain"
)

type memTickets struct {
	mu      sync.Mutex
	created []domain.Ticket
	calls   int
	err     error
}

var ticketTime = time.Date(2026, 11, 25, 14, 30, 0, 0, time.UTC)

func (m *memTickets) Create(_ context.Context, t domain.Ticket) (domain.Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return domain.Ticket{}, m.err
	}
	m.created = append(m.created, t)
	t.ID = domain.TicketID(len(m.created))
	t.Status = domain.TicketStatusOpen
	t.CreatedAt, t.UpdatedAt = ticketTime, ticketTime
	return t, nil
}

func (m *memTickets) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *memTickets) stored() []domain.Ticket {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.Ticket(nil), m.created...)
}

type stubValidator struct {
	ticket domain.Ticket
	err    error
	calls  int
}

func (s *stubValidator) ValidateTicket(domain.TicketDraft) (domain.Ticket, error) {
	s.calls++
	return s.ticket, s.err
}

const author domain.UserID = 42

var errStubInvalid = fmt.Errorf("security: validate ticket: sistema: too short: %w", domain.ErrInvalidInput)

var titleVectors = []struct {
	name  string
	draft domain.TicketDraft
	want  string
}{
	{
		"acesso",
		domain.TicketDraft{Type: domain.TicketTypeAccess, System: "Grafana", AccessLevel: "leitura", Justification: "Preciso ver os painéis."},
		"Acesso a Grafana",
	},
	{
		"acesso, system trimmed",
		domain.TicketDraft{Type: domain.TicketTypeAccess, System: "  Grafana \t", AccessLevel: "escrita", Justification: "Preciso ver os painéis."},
		"Acesso a Grafana",
	},
	{
		"bug",
		domain.TicketDraft{Type: domain.TicketTypeBug, System: "Portal do aluno", Description: "A página não carrega.", Steps: "1. Abrir\n2. Entrar", Severity: "alta"},
		"Bug em Portal do aluno",
	},
	{
		"deploy to dev",
		domain.TicketDraft{Type: domain.TicketTypeDeploy, Project: "api", Repository: "https://git.example/api", Environment: "dev"},
		"Deploy de api em Desenvolvimento",
	},
	{
		"deploy to homolog",
		domain.TicketDraft{Type: domain.TicketTypeDeploy, Project: "api", Repository: "https://git.example/api", Environment: "homolog"},
		"Deploy de api em Homologação",
	},
	{
		"deploy to prod",
		domain.TicketDraft{Type: domain.TicketTypeDeploy, Project: "api", Repository: "https://git.example/api", Environment: "prod"},
		"Deploy de api em Produção",
	},
	{
		"outro keeps the typed title",
		domain.TicketDraft{Type: domain.TicketTypeOther, Title: "  Dúvida sobre a VPN ", Description: "Como peço uma VPN?"},
		"Dúvida sobre a VPN",
	},
	{
		"outro with HTML is kept as typed",
		domain.TicketDraft{Type: domain.TicketTypeOther, Title: "<b>Ajuda</b> & **x**", Description: "Como peço uma VPN?"},
		"<b>Ajuda</b> & **x**",
	},
	{
		"exactly 120 is not cut",
		domain.TicketDraft{Type: domain.TicketTypeBug, System: strings.Repeat("é", 113), Description: "A página não carrega.", Steps: "1. Abrir\n2. Entrar", Severity: "baixa"},
		"Bug em " + strings.Repeat("é", 113),
	},
	{
		"acesso cut to 120",
		domain.TicketDraft{Type: domain.TicketTypeAccess, System: strings.Repeat("é", 120), AccessLevel: "administrador", Justification: "Preciso ver os painéis."},
		"Acesso a " + strings.Repeat("é", 111),
	},
	{
		"emoji at position 120 is kept whole",
		domain.TicketDraft{Type: domain.TicketTypeAccess, System: strings.Repeat("a", 110) + "😀zz", AccessLevel: "leitura", Justification: "Preciso ver os painéis."},
		"Acesso a " + strings.Repeat("a", 110) + "😀",
	},
	{
		"emoji at position 121 is dropped whole",
		domain.TicketDraft{Type: domain.TicketTypeAccess, System: strings.Repeat("a", 111) + "😀", AccessLevel: "leitura", Justification: "Preciso ver os painéis."},
		"Acesso a " + strings.Repeat("a", 111),
	},
	{
		"deploy cut inside the environment label",
		domain.TicketDraft{Type: domain.TicketTypeDeploy, Project: strings.Repeat("p", 105), Repository: "https://git.example/api", Environment: "prod"},
		"Deploy de " + strings.Repeat("p", 105) + " em P",
	},
	{
		"space at the cut is removed",
		domain.TicketDraft{Type: domain.TicketTypeAccess, System: strings.Repeat("a", 110) + " bcd", AccessLevel: "leitura", Justification: "Preciso ver os painéis."},
		"Acesso a " + strings.Repeat("a", 110),
	},
	{
		"several spaces before the cut are removed",
		domain.TicketDraft{Type: domain.TicketTypeBug, System: strings.Repeat("b", 108) + "     xyz", Description: "A página não carrega.", Steps: "1. Abrir\n2. Entrar", Severity: "media"},
		"Bug em " + strings.Repeat("b", 108),
	},
	{
		"no-break space at the cut is removed",
		domain.TicketDraft{Type: domain.TicketTypeAccess, System: strings.Repeat("a", 110) + "\u00a0yy", AccessLevel: "leitura", Justification: "Preciso ver os painéis."},
		"Acesso a " + strings.Repeat("a", 110),
	},
	{
		"deploy cut right after \"em \" drops the space",
		domain.TicketDraft{Type: domain.TicketTypeDeploy, Project: strings.Repeat("p", 106), Repository: "https://git.example/api", Environment: "prod"},
		"Deploy de " + strings.Repeat("p", 106) + " em",
	},
	{
		"inner spaces are kept",
		domain.TicketDraft{Type: domain.TicketTypeOther, Title: "Dúvida   sobre  a VPN", Description: "Como peço uma VPN?"},
		"Dúvida   sobre  a VPN",
	},
	{
		"deploy with a 120 project keeps only the project",
		domain.TicketDraft{Type: domain.TicketTypeDeploy, Project: strings.Repeat("p", 120), Repository: "https://git.example/api", Environment: "dev"},
		"Deploy de " + strings.Repeat("p", 110),
	},
}

func TestCreateTicketTitle(t *testing.T) {
	t.Parallel()

	for _, tt := range titleVectors {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &memTickets{}
			got, err := NewTickets(repo, newSecurity(t)).Create(t.Context(), author, tt.draft)
			if err != nil {
				t.Fatalf("Create() = %v", err)
			}
			if got.Title != tt.want {
				t.Errorf("title = %q, want %q", got.Title, tt.want)
			}
			if n := utf8.RuneCountInString(got.Title); n > maxTitleLen || !utf8.ValidString(got.Title) {
				t.Errorf("title has %d code points (valid UTF-8: %v), want at most %d", n, utf8.ValidString(got.Title), maxTitleLen)
			}
			if stored := repo.stored(); len(stored) != 1 || stored[0].Title != tt.want {
				t.Errorf("stored tickets = %+v, want one titled %q", stored, tt.want)
			}
		})
	}
}

func TestCreateTicket(t *testing.T) {
	t.Parallel()

	repo := &memTickets{}
	draft := domain.TicketDraft{
		Type: domain.TicketTypeDeploy, Project: " api ", Repository: "https://git.example/api",
		Environment: "prod", Notes: new(string),
	}
	*draft.Notes = "  Depois das 18h.\n"

	got, err := NewTickets(repo, newSecurity(t)).Create(t.Context(), author, draft)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	stored := repo.stored()
	if len(stored) != 1 {
		t.Fatalf("stored %d tickets, want 1", len(stored))
	}
	want := domain.Ticket{
		AuthorID: author, Type: domain.TicketTypeDeploy, Status: domain.TicketStatusOpen,
		Title: "Deploy de api em Produção", Project: "api", Repository: "https://git.example/api",
		Environment: domain.EnvironmentProduction, Notes: "Depois das 18h.",
	}
	if stored[0] != want {
		t.Errorf("ticket sent to the repository = %+v, want %+v", stored[0], want)
	}

	want.ID, want.CreatedAt, want.UpdatedAt = 1, ticketTime, ticketTime
	if got != want {
		t.Errorf("Create() = %+v, want %+v", got, want)
	}
}

func TestCreateTicketIgnoresValidatorServerFields(t *testing.T) {
	t.Parallel()

	repo := &memTickets{}
	validator := &stubValidator{ticket: domain.Ticket{
		ID: 99, AuthorID: 7, Status: domain.TicketStatusClosed,
		Type: domain.TicketTypeOther, Title: "Dúvida", Description: "Como peço uma VPN?",
	}}

	if _, err := NewTickets(repo, validator).Create(t.Context(), author, domain.TicketDraft{}); err != nil {
		t.Fatalf("Create() = %v", err)
	}
	stored := repo.stored()
	if len(stored) != 1 {
		t.Fatalf("stored %d tickets, want 1", len(stored))
	}
	if s := stored[0]; s.ID != 0 || s.AuthorID != author || s.Status != domain.TicketStatusOpen {
		t.Errorf("stored id %d, author %d, status %q; want 0, %d, %q", s.ID, s.AuthorID, s.Status, author, domain.TicketStatusOpen)
	}
}

func TestCreateTicketFailures(t *testing.T) {
	t.Parallel()

	const fieldValue = "justificativa-do-usuario"
	tests := []struct {
		name         string
		author       domain.UserID
		draft        domain.TicketDraft
		validator    *stubValidator
		repoErr      error
		want         error
		invalidInput bool
		repoCalled   bool
	}{
		{
			name: "validator error", author: author,
			validator: &stubValidator{err: errStubInvalid},
			want:      errStubInvalid, invalidInput: true,
		},
		{
			name: "invalid field", author: author,
			draft: domain.TicketDraft{Type: domain.TicketTypeAccess, System: "G", AccessLevel: "leitura", Justification: fieldValue},
			want:  domain.ErrInvalidInput, invalidInput: true,
		},
		{
			name: "unknown type", author: author,
			draft: domain.TicketDraft{Type: "incidente", Title: "Dúvida", Description: fieldValue},
			want:  domain.ErrInvalidInput, invalidInput: true,
		},
		{
			name: "field of another type", author: author,
			draft: domain.TicketDraft{Type: domain.TicketTypeOther, Title: "Dúvida", Description: fieldValue, Severity: "alta"},
			want:  domain.ErrInvalidInput, invalidInput: true,
		},
		{
			name: "validated type with no title rule", author: author,
			validator: &stubValidator{ticket: domain.Ticket{Type: "incidente"}},
			want:      domain.ErrInvalidInput, invalidInput: true,
		},
		{
			name: "validated deploy with an unknown environment", author: author,
			validator: &stubValidator{ticket: domain.Ticket{Type: domain.TicketTypeDeploy, Project: "api", Environment: "qa"}},
			want:      domain.ErrInvalidInput, invalidInput: true,
		},
		{
			name: "no author", author: 0,
			draft: domain.TicketDraft{Type: domain.TicketTypeOther, Title: "Dúvida", Description: fieldValue},
			want:  errNoAuthor,
		},
		{
			name: "negative author", author: -1,
			draft: domain.TicketDraft{Type: domain.TicketTypeOther, Title: "Dúvida", Description: fieldValue},
			want:  errNoAuthor,
		},
		{
			name: "repository failure", author: author,
			draft:   domain.TicketDraft{Type: domain.TicketTypeOther, Title: "Dúvida", Description: fieldValue},
			repoErr: errDatabase, want: errDatabase, repoCalled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &memTickets{err: tt.repoErr}
			var validator TicketValidator = newSecurity(t)
			if tt.validator != nil {
				validator = tt.validator
			}

			got, err := NewTickets(repo, validator).Create(t.Context(), tt.author, tt.draft)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Create() = %v, want %v", err, tt.want)
			}
			if !strings.HasPrefix(err.Error(), "service: create ticket: ") {
				t.Errorf("error %q is not wrapped by the service", err)
			}
			if isInvalid := errors.Is(err, domain.ErrInvalidInput); isInvalid != tt.invalidInput {
				t.Errorf("errors.Is(err, ErrInvalidInput) = %v, want %v", isInvalid, tt.invalidInput)
			}
			if strings.Contains(err.Error(), fieldValue) {
				t.Errorf("error %q contains a field value", err)
			}
			if got != (domain.Ticket{}) {
				t.Errorf("Create() returned %+v with an error, want the zero ticket", got)
			}
			if called := repo.callCount() > 0; called != tt.repoCalled {
				t.Errorf("repository called: %v, want %v", called, tt.repoCalled)
			}
			if n := len(repo.stored()); n != 0 {
				t.Errorf("repository stored %d tickets, want 0", n)
			}
		})
	}
}

func TestCreateTicketNoAuthorSkipsValidation(t *testing.T) {
	t.Parallel()

	validator := &stubValidator{}
	repo := &memTickets{}
	if _, err := NewTickets(repo, validator).Create(t.Context(), 0, domain.TicketDraft{}); !errors.Is(err, errNoAuthor) {
		t.Fatalf("Create() = %v, want errNoAuthor", err)
	}
	if validator.calls != 0 || len(repo.stored()) != 0 {
		t.Errorf("validator called %d times and %d tickets stored, want none", validator.calls, len(repo.stored()))
	}
}
