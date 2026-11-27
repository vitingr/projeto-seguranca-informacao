package security

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

func query(t *testing.T, raw string) url.Values {
	t.Helper()
	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("ParseQuery(%q): %v", raw, err)
	}
	return values
}

func wantInvalidQuery(t *testing.T, err error, reason, raw string) {
	t.Helper()
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
	if !strings.Contains(err.Error(), reason) {
		t.Errorf("error %q, want reason %q", err, reason)
	}
	for _, pair := range strings.Split(raw, "&") {
		name, value, _ := strings.Cut(pair, "=")
		for _, text := range []string{name, value} {
			if len(text) > 2 && strings.Contains(err.Error(), text) && !strings.Contains(reason, text) {
				t.Errorf("error %q quotes the client's text %q", err, text)
			}
		}
	}
}

func TestParseTicketListQueryAccepts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		raw   string
		scope TicketListScope
		want  domain.TicketListQuery
	}{
		{"no parameter, own", "", TicketListOwn, domain.TicketListQuery{Page: 1}},
		{"no parameter, queue", "", TicketListQueue, domain.TicketListQuery{Page: 1}},
		{"status aberto", "status=aberto", TicketListOwn, domain.TicketListQuery{Status: domain.TicketStatusOpen, Page: 1}},
		{"status em_andamento", "status=em_andamento", TicketListOwn, domain.TicketListQuery{Status: domain.TicketStatusInProgress, Page: 1}},
		{"status resolvido", "status=resolvido", TicketListQueue, domain.TicketListQuery{Status: domain.TicketStatusResolved, Page: 1}},
		{"status fechado", "status=fechado", TicketListQueue, domain.TicketListQuery{Status: domain.TicketStatusClosed, Page: 1}},
		{"type acesso", "tipo=acesso", TicketListOwn, domain.TicketListQuery{Type: domain.TicketTypeAccess, Page: 1}},
		{"type bug", "tipo=bug", TicketListOwn, domain.TicketListQuery{Type: domain.TicketTypeBug, Page: 1}},
		{"type deploy", "tipo=deploy", TicketListQueue, domain.TicketListQuery{Type: domain.TicketTypeDeploy, Page: 1}},
		{"type outro", "tipo=outro", TicketListQueue, domain.TicketListQuery{Type: domain.TicketTypeOther, Page: 1}},
		{"page 1", "pagina=1", TicketListOwn, domain.TicketListQuery{Page: 1}},
		{"page 2", "pagina=2", TicketListOwn, domain.TicketListQuery{Page: 2}},
		{"page 10000", "pagina=10000", TicketListQueue, domain.TicketListQuery{Page: 10000}},
		{"author in the queue", "usuario=maria.souza", TicketListQueue, domain.TicketListQuery{AuthorLogin: "maria.souza", Page: 1}},
		{
			"every filter, queue", "status=aberto&tipo=bug&usuario=ana_lima-2&pagina=3", TicketListQueue,
			domain.TicketListQuery{Status: domain.TicketStatusOpen, Type: domain.TicketTypeBug, AuthorLogin: "ana_lima-2", Page: 3},
		},
		{
			"every filter, own", "pagina=2&tipo=deploy&status=resolvido", TicketListOwn,
			domain.TicketListQuery{Status: domain.TicketStatusResolved, Type: domain.TicketTypeDeploy, Page: 2},
		},
	}
	s := newTestSecurity(t, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := s.ParseTicketListQuery(query(t, tt.raw), tt.scope)
			if err != nil {
				t.Fatalf("ParseTicketListQuery(%q): %v", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("ParseTicketListQuery(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseTicketListQueryRefuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		scope  TicketListScope
		reason string
	}{
		{"unknown status", "status=cancelado", TicketListOwn, queryInvalidStatus},
		{"status wrong case", "status=Aberto", TicketListOwn, queryInvalidStatus},
		{"status label", "status=Em+andamento", TicketListQueue, queryInvalidStatus},
		{"unknown type", "tipo=incidente", TicketListOwn, queryInvalidType},
		{"type wrong case", "tipo=BUG", TicketListQueue, queryInvalidType},
		{"type with spaces", "tipo=+bug+", TicketListOwn, queryInvalidType},

		{"empty status", "status=", TicketListOwn, queryEmptyValue},
		{"empty type", "tipo=", TicketListQueue, queryEmptyValue},
		{"empty page", "pagina=", TicketListOwn, queryEmptyValue},
		{"empty author in the queue", "usuario=", TicketListQueue, queryEmptyValue},
		{"name without value", "status", TicketListOwn, queryEmptyValue},
		{"repeated status", "status=aberto&status=fechado", TicketListOwn, queryRepeatedParam},
		{"repeated same value", "tipo=bug&tipo=bug", TicketListQueue, queryRepeatedParam},
		{"repeated page", "pagina=1&pagina=2", TicketListOwn, queryRepeatedParam},
		{"unknown name", "ordem=antigos", TicketListOwn, queryUnknownParam},
		{"name in another case", "Status=aberto", TicketListOwn, queryUnknownParam},
		{"page size", "tamanho=100", TicketListQueue, queryUnknownParam},
		{"thread cursor on a list", "antes=10", TicketListQueue, queryUnknownParam},

		{"author in the own list", "usuario=maria.souza", TicketListOwn, queryAuthorRefused},
		{"empty author in the own list", "usuario=", TicketListOwn, queryAuthorRefused},
		{"author in the own list with filters", "status=aberto&usuario=maria.souza", TicketListOwn, queryAuthorRefused},
		{"author too short", "usuario=ab", TicketListQueue, queryInvalidLogin},
		{"author in upper case", "usuario=Maria.Souza", TicketListQueue, queryInvalidLogin},
		{"author with a space", "usuario=maria+souza", TicketListQueue, queryInvalidLogin},
		{"author with a quote", "usuario=maria%27--", TicketListQueue, queryInvalidLogin},
		{"author too long", "usuario=" + strings.Repeat("a", 65), TicketListQueue, queryInvalidLogin},

		{"page 0", "pagina=0", TicketListOwn, queryInvalidPage},
		{"page 10001", "pagina=10001", TicketListOwn, queryInvalidPage},
		{"page -1", "pagina=-1", TicketListQueue, queryInvalidPage},
		{"page +1", "pagina=%2B1", TicketListOwn, queryInvalidPage},
		{"page 01", "pagina=01", TicketListOwn, queryInvalidPage},
		{"page 1.0", "pagina=1.0", TicketListQueue, queryInvalidPage},
		{"page abc", "pagina=abc", TicketListOwn, queryInvalidPage},
		{"page 1e3", "pagina=1e3", TicketListOwn, queryInvalidPage},
		{"page past int64", "pagina=9223372036854775808", TicketListOwn, queryInvalidPage},
	}
	s := newTestSecurity(t, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := s.ParseTicketListQuery(query(t, tt.raw), tt.scope)
			wantInvalidQuery(t, err, tt.reason, tt.raw)
			if got != (domain.TicketListQuery{}) {
				t.Errorf("ParseTicketListQuery(%q) = %+v with an error, want the zero query", tt.raw, got)
			}
		})
	}
}

func TestParseTicketListQueryInvalidScope(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	for _, scope := range []TicketListScope{0, 3, -1} {
		_, err := s.ParseTicketListQuery(url.Values{}, scope)
		if !errors.Is(err, errInvalidScope) || errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("scope %d: error = %v, want errInvalidScope (a 500, not a 400)", scope, err)
		}
	}
}

func TestParseThreadCursor(t *testing.T) {
	t.Parallel()

	accepted := []struct {
		name string
		raw  string
		want domain.TicketEntryID
	}{
		{"absent: the latest entries", "", 0},
		{"one", "antes=1", 1},
		{"an entry id", "antes=1234", 1234},
		{"largest int64", "antes=9223372036854775807", 9223372036854775807},
	}
	s := newTestSecurity(t, nil)
	for _, tt := range accepted {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := s.ParseThreadCursor(query(t, tt.raw))
			if err != nil || got != tt.want {
				t.Errorf("ParseThreadCursor(%q) = %d, %v; want %d, nil", tt.raw, got, err, tt.want)
			}
		})
	}

	refused := []struct {
		name   string
		raw    string
		reason string
	}{
		{"zero", "antes=0", queryInvalidCursor},
		{"negative", "antes=-1", queryInvalidCursor},
		{"plus sign", "antes=%2B1", queryInvalidCursor},
		{"leading zero", "antes=01", queryInvalidCursor},
		{"exponent", "antes=1e3", queryInvalidCursor},
		{"letters", "antes=abc", queryInvalidCursor},
		{"leading space", "antes=+1", queryInvalidCursor},
		{"one past int64", "antes=9223372036854775808", queryInvalidCursor},
		{"non-ASCII digit", "antes=%D9%A1", queryInvalidCursor},
		{"empty", "antes=", queryEmptyValue},
		{"repeated", "antes=1&antes=2", queryRepeatedParam},
		{"unknown name", "depois=1", queryUnknownParam},
		{"list parameter", "pagina=2", queryUnknownParam},
	}
	for _, tt := range refused {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := s.ParseThreadCursor(query(t, tt.raw))
			wantInvalidQuery(t, err, tt.reason, tt.raw)
			if got != 0 {
				t.Errorf("ParseThreadCursor(%q) = %d with an error, want 0", tt.raw, got)
			}
		})
	}
}

func TestReadQuery(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		name    string
		target  string
		want    url.Values
		wantErr bool
	}{
		{"no query", "/api/tickets", url.Values{}, false},
		{"pairs", "/api/tickets?status=aberto&pagina=2", url.Values{"status": {"aberto"}, "pagina": {"2"}}, false},
		{"semicolon", "/api/tickets?status=aberto;pagina=2", nil, true},
		{"bad escape", "/api/tickets?status=%zz", nil, true},
		{"bad escape in a name", "/api/tickets?%zz=1", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.target, http.NoBody)
			got, err := s.ReadQuery(req)
			if tt.wantErr {
				wantInvalidQuery(t, err, queryMalformed, "")
				return
			}
			if err != nil {
				t.Fatalf("ReadQuery(%q): %v", tt.target, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ReadQuery(%q) = %v, want %v", tt.target, got, tt.want)
			}
			for name, vs := range tt.want {
				if len(got[name]) != len(vs) || got.Get(name) != vs[0] {
					t.Errorf("ReadQuery(%q)[%q] = %v, want %v", tt.target, name, got[name], vs)
				}
			}
		})
	}
}
