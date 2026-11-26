package security

import (
	"errors"
	"io"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"backend_si/internal/domain"
)

// rep returns s repeated n times.
func rep(s string, n int) string { return strings.Repeat(s, n) }

type fieldCase struct {
	name   string
	raw    string
	want   string // stored value when the rule accepts it
	reason string // reason in the error; empty = accepted
}

func runFieldCases(t *testing.T, rule fieldRule, tests []fieldCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := rule(tt.raw)
			if tt.reason == "" {
				if err != nil {
					t.Fatalf("error = %v, want accepted", err)
				}
				if got != tt.want {
					t.Errorf("value = %q, want %q", got, tt.want)
				}
				return
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
			if !strings.Contains(err.Error(), tt.reason) {
				t.Errorf("error = %q, want reason %q", err, tt.reason)
			}
			if got != "" {
				t.Errorf("value = %q on error, want empty", got)
			}
			if trimmed := strings.TrimSpace(tt.raw); len(trimmed) >= 4 && strings.Contains(err.Error(), trimmed) {
				t.Errorf("error %q leaks the value", err)
			}
		})
	}
}

func TestSingleLine(t *testing.T) {
	t.Parallel()

	runFieldCases(t, singleLine(2, 120), []fieldCase{
		{"plain", "Grafana", "Grafana", ""},
		{"trimmed", "  \t Grafana \n ", "Grafana", ""},
		{"inner spaces kept", "Portal  do aluno", "Portal  do aluno", ""},
		{"html kept as typed", "<b>x</b>", "<b>x</b>", ""},
		{"empty", "", "", fieldEmpty},
		{"spaces only", " \t\n\r ", "", fieldEmpty},
		{"min - 1", "a", "", fieldTooShort},
		{"min", "ab", "ab", ""},
		{"min after trim - 1", "  a  ", "", fieldTooShort},
		{"max", rep("a", 120), rep("a", 120), ""},
		{"max + 1", rep("a", 121), "", fieldTooLong},
		{"multi-byte at the max", rep("é", 119) + "😀", rep("é", 119) + "😀", ""},
		{"multi-byte past the max", rep("é", 120) + "😀", "", fieldTooLong},
		{"line feed", "Gra\nfana", "", fieldLineBreak},
		{"carriage return", "Gra\rfana", "", fieldLineBreak},
		{"line separator", "Gra fana", "", fieldLineBreak},
		{"paragraph separator", "Gra fana", "", fieldLineBreak},
		{"tab", "Gra\tfana", "", fieldControl},
		{"NUL", "Gra\x00fana", "", fieldControl},
		{"escape", "Gra\x1bfana", "", fieldControl},
		{"DEL", "Gra\x7ffana", "", fieldControl},
		{"C1 control", "Gra\u0085fana", "", fieldControl},
		{"invalid utf-8", "Gra\xfffana", "", fieldInvalidUTF8},
		{"truncated utf-8", "Grafana\xc3", "", fieldInvalidUTF8},
	})
}

func TestMultiLine(t *testing.T) {
	t.Parallel()

	runFieldCases(t, multiLine(10, 2000), []fieldCase{
		{"plain", "Preciso ver os painéis.", "Preciso ver os painéis.", ""},
		{"line breaks and tabs kept", "1. Abrir\r\n2. Entrar\n\tdepois", "1. Abrir\r\n2. Entrar\n\tdepois", ""},
		{"trimmed", "\n\n  Preciso ver os painéis.  \n", "Preciso ver os painéis.", ""},
		{"markdown kept as typed", "**negrito** e _x_", "**negrito** e _x_", ""},
		{"empty", "", "", fieldEmpty},
		{"spaces and breaks only", " \n\t\r\n ", "", fieldEmpty},
		{"min - 1", rep("a", 9), "", fieldTooShort},
		{"min", rep("a", 10), rep("a", 10), ""},
		{"max", rep("a", 2000), rep("a", 2000), ""},
		{"max + 1", rep("a", 2001), "", fieldTooLong},
		{"multi-byte at the max", rep("ç", 1999) + "😀", rep("ç", 1999) + "😀", ""},
		{"multi-byte past the max", rep("ç", 2000) + "😀", "", fieldTooLong},
		{"line separator allowed", "linha um linha dois", "linha um linha dois", ""},
		{"NUL", "Preciso ver\x00 os painéis", "", fieldControl},
		{"escape", "Preciso ver\x1b[31m os painéis", "", fieldControl},
		{"vertical tab", "Preciso ver\v os painéis", "", fieldControl},
		{"form feed", "Preciso ver\f os painéis", "", fieldControl},
		{"DEL", "Preciso ver\x7f os painéis", "", fieldControl},
		{"C1 control", "Preciso ver\u0085 os painéis", "", fieldControl},
		{"invalid utf-8", "Preciso ver \xff os painéis", "", fieldInvalidUTF8},
	})
}

func TestMultiLineOptional(t *testing.T) {
	t.Parallel()

	runFieldCases(t, multiLine(0, 2000), []fieldCase{
		{"empty is allowed", "", "", ""},
		{"spaces only become empty", "  \n ", "", ""},
		{"one character", "x", "x", ""},
		{"max", rep("a", 2000), rep("a", 2000), ""},
		{"max + 1", rep("a", 2001), "", fieldTooLong},
		{"NUL still refused", "a\x00b", "", fieldControl},
	})
}

func TestHTTPSURL(t *testing.T) {
	t.Parallel()

	const base = "https://git.example.com/"
	runFieldCases(t, httpsURL(300), []fieldCase{
		{"repository", "https://github.com/org/repo", "https://github.com/org/repo", ""},
		{"with .git, query and fragment", "https://git.example.com/a/b.git?ref=main#readme", "https://git.example.com/a/b.git?ref=main#readme", ""},
		{"with port", "https://git.example.com:8443/a", "https://git.example.com:8443/a", ""},
		{"ip host", "https://10.0.0.1/repo", "https://10.0.0.1/repo", ""},
		{"upper case scheme", "HTTPS://git.example.com/a", "HTTPS://git.example.com/a", ""},
		{"trimmed", "  https://github.com/org/repo \n", "https://github.com/org/repo", ""},
		{"max", base + rep("a", 300-len(base)), base + rep("a", 300-len(base)), ""},
		{"max + 1", base + rep("a", 301-len(base)), "", fieldTooLong},
		{"multi-byte at the max", base + rep("é", 300-len(base)), base + rep("é", 300-len(base)), ""},
		{"empty", "", "", fieldEmpty},
		{"spaces only", "   ", "", fieldEmpty},
		{"http", "http://github.com/org/repo", "", fieldNotHTTPSURL},
		{"javascript", "javascript:alert(1)", "", fieldNotHTTPSURL},
		{"data", "data:text/html,<script>alert(1)</script>", "", fieldNotHTTPSURL},
		{"ssh scp form", "git@github.com:org/repo.git", "", fieldNotHTTPSURL},
		{"ssh url", "ssh://git@github.com/org/repo.git", "", fieldNotHTTPSURL},
		{"no scheme", "github.com/org/repo", "", fieldNotHTTPSURL},
		{"scheme relative", "//github.com/org/repo", "", fieldNotHTTPSURL},
		{"no host", "https://", "", fieldNotHTTPSURL},
		{"no host with path", "https:///org/repo", "", fieldNotHTTPSURL},
		{"opaque", "https:github.com/org/repo", "", fieldNotHTTPSURL},
		{"port only", "https://:443/repo", "", fieldNotHTTPSURL},
		{"user and password", "https://user:pass@github.com/org/repo", "", fieldCredentials},
		{"user only", "https://token@github.com/org/repo", "", fieldCredentials},
		{"inner space", "https://github.com/org/my repo", "", fieldNotHTTPSURL},
		{"bad port", "https://github.com:abc/org", "", fieldNotHTTPSURL},
		{"line feed", "https://github.com/org\n/repo", "", fieldLineBreak},
		{"NUL", "https://github.com/\x00repo", "", fieldControl},
		{"invalid utf-8", "https://github.com/\xffrepo", "", fieldInvalidUTF8},
	})
}

// marker is in every value of the ticket tests, so they can check that no
// error ever carries a field value (S-7).
const marker = "marcador7f3a"

func strPtr(s string) *string { return &s }

func validDraft(typ domain.TicketType) domain.TicketDraft {
	switch typ {
	case domain.TicketTypeAccess:
		return domain.TicketDraft{Type: typ, System: " Grafana " + marker, AccessLevel: "leitura", Justification: "\nPreciso ver os painéis " + marker + ".\n"}
	case domain.TicketTypeBug:
		return domain.TicketDraft{Type: typ, System: "Portal " + marker, Description: "A página não carrega " + marker, Steps: "1. Abrir\n2. Entrar " + marker, Severity: "critica"}
	case domain.TicketTypeDeploy:
		return domain.TicketDraft{Type: typ, Project: "api-" + marker, Repository: " https://git.example.com/org/" + marker + " ", Environment: "prod"}
	default:
		return domain.TicketDraft{Type: typ, Title: "  Dúvida " + marker, Description: "Como peço uma VPN? " + marker}
	}
}

func TestValidateTicketAccepts(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	withNotes := func(notes *string) domain.TicketDraft {
		d := validDraft(domain.TicketTypeDeploy)
		d.Notes = notes
		return d
	}
	tests := []struct {
		name  string
		draft domain.TicketDraft
		want  domain.Ticket
	}{
		{"acesso", validDraft(domain.TicketTypeAccess), domain.Ticket{
			Type: domain.TicketTypeAccess, System: "Grafana " + marker, AccessLevel: domain.AccessLevelRead,
			Justification: "Preciso ver os painéis " + marker + ".",
		}},
		{"bug", validDraft(domain.TicketTypeBug), domain.Ticket{
			Type: domain.TicketTypeBug, System: "Portal " + marker, Description: "A página não carrega " + marker,
			Steps: "1. Abrir\n2. Entrar " + marker, Severity: domain.SeverityCritical,
		}},
		{"deploy without notes", withNotes(nil), domain.Ticket{
			Type: domain.TicketTypeDeploy, Project: "api-" + marker,
			Repository: "https://git.example.com/org/" + marker, Environment: domain.EnvironmentProduction,
		}},
		{"deploy with empty notes", withNotes(strPtr("")), domain.Ticket{
			Type: domain.TicketTypeDeploy, Project: "api-" + marker,
			Repository: "https://git.example.com/org/" + marker, Environment: domain.EnvironmentProduction,
		}},
		{"deploy with blank notes", withNotes(strPtr(" \n ")), domain.Ticket{
			Type: domain.TicketTypeDeploy, Project: "api-" + marker,
			Repository: "https://git.example.com/org/" + marker, Environment: domain.EnvironmentProduction,
		}},
		{"deploy with notes", withNotes(strPtr(" Depois das 18h. ")), domain.Ticket{
			Type: domain.TicketTypeDeploy, Project: "api-" + marker,
			Repository: "https://git.example.com/org/" + marker, Environment: domain.EnvironmentProduction,
			Notes: "Depois das 18h.",
		}},
		{"outro", validDraft(domain.TicketTypeOther), domain.Ticket{
			Type: domain.TicketTypeOther, Title: "Dúvida " + marker, Description: "Como peço uma VPN? " + marker,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := s.ValidateTicket(tt.draft)
			if err != nil {
				t.Fatalf("ValidateTicket: %v", err)
			}
			if got != tt.want {
				t.Errorf("ticket = %+v\nwant     %+v", got, tt.want)
			}
		})
	}
}

func TestValidateTicketRefuses(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	with := func(typ domain.TicketType, change func(*domain.TicketDraft)) domain.TicketDraft {
		d := validDraft(typ)
		change(&d)
		return d
	}
	const (
		access = domain.TicketTypeAccess
		bug    = domain.TicketTypeBug
		deploy = domain.TicketTypeDeploy
		other  = domain.TicketTypeOther
	)
	tests := []struct {
		name   string
		draft  domain.TicketDraft
		reason string // field name or rule reason expected in the error
	}{
		// Type.
		{"unknown type", with(other, func(d *domain.TicketDraft) { d.Type = "incidente" }), "unknown type"},
		{"empty type", with(other, func(d *domain.TicketDraft) { d.Type = "" }), "unknown type"},
		{"type in upper case", with(other, func(d *domain.TicketDraft) { d.Type = "OUTRO" }), "unknown type"},

		// A field of another type.
		{"acesso with descricao", with(access, func(d *domain.TicketDraft) { d.Description = "Descrição " + marker }), "descricao: not a field"},
		{"bug with titulo", with(bug, func(d *domain.TicketDraft) { d.Title = "Título " + marker }), "titulo: not a field"},
		{"bug with empty notes", with(bug, func(d *domain.TicketDraft) { d.Notes = strPtr("") }), "observacoes: not a field"},
		{"deploy with sistema", with(deploy, func(d *domain.TicketDraft) { d.System = "Portal " + marker }), "sistema: not a field"},
		{"outro with severidade", with(other, func(d *domain.TicketDraft) { d.Severity = "baixa" }), "severidade: not a field"},

		// acesso.
		{"acesso sistema missing", with(access, func(d *domain.TicketDraft) { d.System = "" }), "sistema: empty"},
		{"acesso sistema spaces only", with(access, func(d *domain.TicketDraft) { d.System = "   " }), "sistema: empty"},
		{"acesso sistema too short", with(access, func(d *domain.TicketDraft) { d.System = "é" }), "sistema: too short"},
		{"acesso sistema too long", with(access, func(d *domain.TicketDraft) { d.System = rep(marker, 11) }), "sistema: too long"},
		{"acesso sistema line break", with(access, func(d *domain.TicketDraft) { d.System = "Gra\nfana " + marker }), "sistema: line break"},
		{"acesso nivel_acesso missing", with(access, func(d *domain.TicketDraft) { d.AccessLevel = "" }), "nivel_acesso: not one of"},
		{"acesso nivel_acesso label", with(access, func(d *domain.TicketDraft) { d.AccessLevel = "Leitura" }), "nivel_acesso: not one of"},
		{"acesso nivel_acesso padded", with(access, func(d *domain.TicketDraft) { d.AccessLevel = " leitura" }), "nivel_acesso: not one of"},
		{"acesso nivel_acesso unknown", with(access, func(d *domain.TicketDraft) { d.AccessLevel = "root" }), "nivel_acesso: not one of"},
		{"acesso justificativa missing", with(access, func(d *domain.TicketDraft) { d.Justification = "" }), "justificativa: empty"},
		{"acesso justificativa too short", with(access, func(d *domain.TicketDraft) { d.Justification = rep("é", 9) }), "justificativa: too short"},
		{"acesso justificativa too long", with(access, func(d *domain.TicketDraft) { d.Justification = rep("a", 2001) }), "justificativa: too long"},
		{"acesso justificativa NUL", with(access, func(d *domain.TicketDraft) { d.Justification = "Preciso\x00 " + marker }), "justificativa: control"},

		// bug.
		{"bug sistema missing", with(bug, func(d *domain.TicketDraft) { d.System = "" }), "sistema: empty"},
		{"bug descricao too short", with(bug, func(d *domain.TicketDraft) { d.Description = "curta" }), "descricao: too short"},
		{"bug descricao invalid utf-8", with(bug, func(d *domain.TicketDraft) { d.Description = "Não \xff carrega " + marker }), "descricao: invalid utf-8"},
		{"bug passos missing", with(bug, func(d *domain.TicketDraft) { d.Steps = " \n " }), "passos: empty"},
		{"bug passos too long", with(bug, func(d *domain.TicketDraft) { d.Steps = rep("ç", 2001) }), "passos: too long"},
		{"bug passos escape", with(bug, func(d *domain.TicketDraft) { d.Steps = "1. Abrir\x1b[2J " + marker }), "passos: control"},
		{"bug severidade missing", with(bug, func(d *domain.TicketDraft) { d.Severity = "" }), "severidade: not one of"},
		{"bug severidade accented label", with(bug, func(d *domain.TicketDraft) { d.Severity = "média" }), "severidade: not one of"},
		{"bug severidade wrong case", with(bug, func(d *domain.TicketDraft) { d.Severity = "Alta" }), "severidade: not one of"},

		// deploy.
		{"deploy projeto missing", with(deploy, func(d *domain.TicketDraft) { d.Project = "" }), "projeto: empty"},
		{"deploy projeto tab", with(deploy, func(d *domain.TicketDraft) { d.Project = "api\t" + marker }), "projeto: control"},
		{"deploy projeto too long", with(deploy, func(d *domain.TicketDraft) { d.Project = rep("a", 121) }), "projeto: too long"},
		{"deploy repositorio missing", with(deploy, func(d *domain.TicketDraft) { d.Repository = "" }), "repositorio: empty"},
		{"deploy repositorio http", with(deploy, func(d *domain.TicketDraft) { d.Repository = "http://git.example.com/" + marker }), "repositorio: not an https url"},
		{"deploy repositorio javascript", with(deploy, func(d *domain.TicketDraft) { d.Repository = "javascript:alert('" + marker + "')" }), "repositorio: not an https url"},
		{"deploy repositorio ssh", with(deploy, func(d *domain.TicketDraft) { d.Repository = "git@github.com:org/" + marker + ".git" }), "repositorio: not an https url"},
		{"deploy repositorio no host", with(deploy, func(d *domain.TicketDraft) { d.Repository = "https:///" + marker }), "repositorio: not an https url"},
		{"deploy repositorio credentials", with(deploy, func(d *domain.TicketDraft) { d.Repository = "https://user:" + marker + "@git.example.com/x" }), "repositorio: credentials"},
		{"deploy repositorio too long", with(deploy, func(d *domain.TicketDraft) { d.Repository = "https://git.example.com/" + rep("a", 277) }), "repositorio: too long"},
		{"deploy ambiente missing", with(deploy, func(d *domain.TicketDraft) { d.Environment = "" }), "ambiente: not one of"},
		{"deploy ambiente label", with(deploy, func(d *domain.TicketDraft) { d.Environment = "Produção" }), "ambiente: not one of"},
		{"deploy observacoes too long", with(deploy, func(d *domain.TicketDraft) { d.Notes = strPtr(rep("a", 2001)) }), "observacoes: too long"},
		{"deploy observacoes NUL", with(deploy, func(d *domain.TicketDraft) { d.Notes = strPtr("Depois\x00 " + marker) }), "observacoes: control"},

		// outro.
		{"outro titulo missing", with(other, func(d *domain.TicketDraft) { d.Title = "" }), "titulo: empty"},
		{"outro titulo too short", with(other, func(d *domain.TicketDraft) { d.Title = " a " }), "titulo: too short"},
		{"outro titulo too long", with(other, func(d *domain.TicketDraft) { d.Title = rep("é", 121) }), "titulo: too long"},
		{"outro titulo line break", with(other, func(d *domain.TicketDraft) { d.Title = "Dúvida\r\n" + marker }), "titulo: line break"},
		{"outro descricao missing", with(other, func(d *domain.TicketDraft) { d.Description = "" }), "descricao: empty"},
		{"outro descricao too long", with(other, func(d *domain.TicketDraft) { d.Description = rep("a", 2001) }), "descricao: too long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := s.ValidateTicket(tt.draft)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
			if !strings.Contains(err.Error(), tt.reason) {
				t.Errorf("error = %q, want %q in it", err, tt.reason)
			}
			if strings.Contains(err.Error(), marker) {
				t.Errorf("error %q leaks a field value", err)
			}
			if got != (domain.Ticket{}) {
				t.Errorf("ticket = %+v on error, want zero", got)
			}
		})
	}
}

func decodeTicket(t *testing.T, s *Security, contentType, body string) (domain.TicketDraft, error) {
	t.Helper()
	req := newJSONRequest(t, contentType, body)
	return s.DecodeTicket(httptest.NewRecorder(), req)
}

func TestDecodeTicketAccepts(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	tests := []struct {
		name string
		body string
		want domain.TicketDraft
	}{
		{
			"acesso", `{"tipo":"acesso","dados":{"sistema":" Grafana ","nivel_acesso":"leitura","justificativa":"Preciso ver.\n"}}`,
			domain.TicketDraft{Type: "acesso", System: " Grafana ", AccessLevel: "leitura", Justification: "Preciso ver.\n"},
		},
		{
			"bug", `{"tipo":"bug","dados":{"sistema":"Portal","descricao":"Não carrega","passos":"1. Abrir","severidade":"alta"}}`,
			domain.TicketDraft{Type: "bug", System: "Portal", Description: "Não carrega", Steps: "1. Abrir", Severity: "alta"},
		},
		{
			"deploy without observacoes", `{"tipo":"deploy","dados":{"projeto":"api","repositorio":"https://git.example.com/api","ambiente":"prod"}}`,
			domain.TicketDraft{Type: "deploy", Project: "api", Repository: "https://git.example.com/api", Environment: "prod"},
		},
		{
			"deploy with empty observacoes", `{"tipo":"deploy","dados":{"projeto":"api","repositorio":"https://git.example.com/api","ambiente":"prod","observacoes":""}}`,
			domain.TicketDraft{Type: "deploy", Project: "api", Repository: "https://git.example.com/api", Environment: "prod", Notes: strPtr("")},
		},
		{
			"deploy with null observacoes", `{"tipo":"deploy","dados":{"projeto":"api","repositorio":"https://git.example.com/api","ambiente":"prod","observacoes":null}}`,
			domain.TicketDraft{Type: "deploy", Project: "api", Repository: "https://git.example.com/api", Environment: "prod"},
		},
		{
			"outro", `{"dados":{"descricao":"Como peço uma VPN?","titulo":"Dúvida"},"tipo":"outro"}`,
			domain.TicketDraft{Type: "outro", Title: "Dúvida", Description: "Como peço uma VPN?"},
		},
		// Left to ValidateTicket (422): missing keys decode as empty, and an
		// unknown or missing tipo decodes with no field.
		{
			"missing required key", `{"tipo":"outro","dados":{"titulo":"Dúvida"}}`,
			domain.TicketDraft{Type: "outro", Title: "Dúvida"},
		},
		{"empty dados", `{"tipo":"bug","dados":{}}`, domain.TicketDraft{Type: "bug"}},
		{"unknown tipo", `{"tipo":"incidente","dados":{"qualquer":"coisa"}}`, domain.TicketDraft{Type: "incidente"}},
		{"missing tipo", `{"dados":{"titulo":"Dúvida"}}`, domain.TicketDraft{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeTicket(t, s, "application/json", tt.body)
			if err != nil {
				t.Fatalf("DecodeTicket: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("draft = %+v\nwant    %+v", got, tt.want)
			}
		})
	}
}

func TestDecodeTicketRefuses(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	const (
		ct     = "application/json"
		access = `"tipo":"acesso","dados":{"sistema":"` + marker + `","nivel_acesso":"leitura","justificativa":"` + marker + `"`
	)
	valid := `{` + access + `}}`
	tests := []struct {
		name        string
		contentType string
		body        string
		reason      string
	}{
		// Keys of another type, unknown keys.
		{"field of another type", ct, `{` + access + `,"descricao":"` + marker + `"}}`, reasonMalformed},
		{"notes in a bug", ct, `{"tipo":"bug","dados":{"observacoes":"` + marker + `"}}`, reasonMalformed},
		{"unknown field in dados", ct, `{` + access + `,"` + marker + `":"x"}}`, reasonMalformed},
		{"unknown field outside", ct, `{` + access + `},"` + marker + `":"x"}`, reasonMalformed},
		{"type key in english", ct, `{"type":"outro","dados":{"titulo":"` + marker + `"}}`, reasonMalformed},

		// Server-owned keys (S-4, S-5), outside and inside dados.
		{"author outside", ct, `{` + access + `},"author":"` + marker + `"}`, reasonMalformed},
		{"user_id outside", ct, `{` + access + `},"user_id":1}`, reasonMalformed},
		{"author_id outside", ct, `{` + access + `},"author_id":1}`, reasonMalformed},
		{"status outside", ct, `{` + access + `},"status":"resolvido"}`, reasonMalformed},
		{"id outside", ct, `{` + access + `},"id":1}`, reasonMalformed},
		{"created_at outside", ct, `{` + access + `},"created_at":"2020-01-01T00:00:00Z"}`, reasonMalformed},
		{"author inside", ct, `{` + access + `,"author":"` + marker + `"}}`, reasonMalformed},
		{"user_id inside", ct, `{` + access + `,"user_id":1}}`, reasonMalformed},
		{"status inside", ct, `{` + access + `,"status":"resolvido"}}`, reasonMalformed},
		{"id inside", ct, `{` + access + `,"id":1}}`, reasonMalformed},
		{"created_at inside", ct, `{` + access + `,"created_at":"2020-01-01T00:00:00Z"}}`, reasonMalformed},
		{"updated_at inside", ct, `{` + access + `,"updated_at":"2020-01-01T00:00:00Z"}}`, reasonMalformed},

		// dados missing or not an object.
		{"dados missing", ct, `{"tipo":"outro"}`, reasonNoData},
		{"dados null", ct, `{"tipo":"outro","dados":null}`, reasonNoData},
		{"dados array", ct, `{"tipo":"outro","dados":["` + marker + `"]}`, reasonMalformed},
		{"dados string", ct, `{"tipo":"outro","dados":"` + marker + `"}`, reasonMalformed},
		{"dados array with unknown tipo", ct, `{"tipo":"incidente","dados":["` + marker + `"]}`, reasonMalformed},
		{"dados number with unknown tipo", ct, `{"tipo":"incidente","dados":1}`, reasonMalformed},

		// Wrong JSON types.
		{"tipo number", ct, `{"tipo":1,"dados":{}}`, reasonMalformed},
		{"sistema number", ct, `{"tipo":"acesso","dados":{"sistema":1}}`, reasonMalformed},
		{"sistema object", ct, `{"tipo":"acesso","dados":{"sistema":{"x":"` + marker + `"}}}`, reasonMalformed},
		{"observacoes array", ct, `{"tipo":"deploy","dados":{"observacoes":["` + marker + `"]}}`, reasonMalformed},
		{"severidade bool", ct, `{"tipo":"bug","dados":{"severidade":true}}`, reasonMalformed},
		{"outer array", ct, `[` + valid + `]`, reasonMalformed},

		// The envelope rules of DecodeJSON.
		{"trailing data", ct, valid + `{"x":"` + marker + `"}`, reasonTrailing},
		{"wrong content type", "text/plain", valid, reasonContentType},
		{"no content type", "", valid, reasonContentType},
		{"empty body", ct, ``, reasonEmpty},
		{"body over 64 KiB", ct, `{"tipo":"outro","dados":{"descricao":"` + rep(marker, MaxBodyBytes/len(marker)+1) + `"}}`, reasonTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeTicket(t, s, tt.contentType, tt.body)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
			if !strings.Contains(err.Error(), tt.reason) {
				t.Errorf("error = %q, want reason %q", err, tt.reason)
			}
			if strings.Contains(err.Error(), marker) {
				t.Errorf("error %q quotes the body", err)
			}
			if !reflect.DeepEqual(got, domain.TicketDraft{}) {
				t.Errorf("draft = %+v on error, want zero", got)
			}
		})
	}
}

// TestDecodeTicketLargestValid sends the largest valid ticket (a bug: two
// 2000-character fields and a 120-character one) with every character an
// emoji written as a JSON escape pair, 12 bytes each, the most a client can
// spend per character. It must fit the 64 KiB body limit and validate.
func TestDecodeTicketLargestValid(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	const emoji = `😀`
	body := `{"tipo":"bug","dados":{"sistema":"` + rep(emoji, 120) +
		`","descricao":"` + rep(emoji, 2000) +
		`","passos":"` + rep(emoji, 2000) +
		`","severidade":"critica"}}`
	if len(body) > MaxBodyBytes {
		t.Fatalf("body is %d bytes, over the %d limit", len(body), MaxBodyBytes)
	}
	draft, err := decodeTicket(t, s, "application/json", body)
	if err != nil {
		t.Fatalf("DecodeTicket: %v", err)
	}
	ticket, err := s.ValidateTicket(draft)
	if err != nil {
		t.Fatalf("ValidateTicket: %v", err)
	}
	if got := utf8.RuneCountInString(ticket.Steps); got != 2000 {
		t.Errorf("steps has %d characters, want 2000", got)
	}
}

// TestDecodeTicketKeySpelling covers plan P-10: encoding/json alone matches
// keys in any case and keeps the last duplicate, so a server-owned key could
// get in as "AUTHOR", and {"tipo":…,"TIPO":…} would silently switch types.
// Keys must be spelled exactly, and none may repeat, even in another case.
func TestDecodeTicketKeySpelling(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	const (
		fields = `"sistema":"` + marker + `","nivel_acesso":"leitura","justificativa":"` + marker + `"`
		data   = `"dados":{` + fields + `}`
	)
	tests := []struct {
		name   string
		body   string
		reason string
	}{
		// Server-owned keys in another case, outside and inside dados.
		{"AUTHOR outside", `{"tipo":"acesso",` + data + `,"AUTHOR":"` + marker + `"}`, reasonMalformed},
		{"Status outside", `{"tipo":"acesso",` + data + `,"Status":"resolvido"}`, reasonMalformed},
		{"User_Id outside", `{"tipo":"acesso",` + data + `,"User_Id":1}`, reasonMalformed},
		{"ID outside", `{"tipo":"acesso",` + data + `,"ID":1}`, reasonMalformed},
		{"Created_At outside", `{"tipo":"acesso",` + data + `,"Created_At":"2020-01-01T00:00:00Z"}`, reasonMalformed},
		{"AUTHOR inside", `{"tipo":"acesso","dados":{` + fields + `,"AUTHOR":"` + marker + `"}}`, reasonMalformed},
		{"Status inside", `{"tipo":"acesso","dados":{` + fields + `,"Status":"resolvido"}}`, reasonMalformed},
		{"User_Id inside", `{"tipo":"acesso","dados":{` + fields + `,"User_Id":1}}`, reasonMalformed},
		{"ID inside", `{"tipo":"acesso","dados":{` + fields + `,"ID":1}}`, reasonMalformed},

		// Documented keys in another case.
		{"Tipo", `{"Tipo":"acesso",` + data + `}`, reasonMalformed},
		{"DADOS", `{"tipo":"acesso","DADOS":{` + fields + `}}`, reasonMalformed},
		{"Sistema", `{"tipo":"acesso","dados":{"Sistema":"` + marker + `","nivel_acesso":"leitura","justificativa":"` + marker + `"}}`, reasonMalformed},
		{"NIVEL_ACESSO", `{"tipo":"acesso","dados":{"sistema":"` + marker + `","NIVEL_ACESSO":"leitura","justificativa":"` + marker + `"}}`, reasonMalformed},
		{"Observacoes", `{"tipo":"deploy","dados":{"projeto":"api","repositorio":"https://git.example.com/a","ambiente":"dev","Observacoes":"` + marker + `"}}`, reasonMalformed},

		// Duplicate keys, exact or in another case.
		{"tipo and TIPO", `{"tipo":"acesso",` + data + `,"TIPO":"outro"}`, reasonDuplicateKey},
		// Refused at its first key already, for the spelling.
		{"TIPO before tipo", `{"TIPO":"outro","tipo":"acesso",` + data + `}`, reasonMalformed},
		{"tipo twice", `{"tipo":"outro","tipo":"acesso",` + data + `}`, reasonDuplicateKey},
		{"dados twice", `{"tipo":"acesso",` + data + `,"dados":{"sistema":"x"}}`, reasonDuplicateKey},
		{"sistema twice", `{"tipo":"acesso","dados":{` + fields + `,"sistema":"` + marker + `"}}`, reasonDuplicateKey},
		{"sistema and SISTEMA", `{"tipo":"acesso","dados":{` + fields + `,"SISTEMA":"` + marker + `"}}`, reasonDuplicateKey},
		{"duplicate in dados of an unknown tipo", `{"tipo":"incidente","dados":{"a":"` + marker + `","A":"x"}}`, reasonDuplicateKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeTicket(t, s, "application/json", tt.body)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput (400)", err)
			}
			if !strings.Contains(err.Error(), tt.reason) {
				t.Errorf("error = %q, want reason %q", err, tt.reason)
			}
			if strings.Contains(err.Error(), marker) {
				t.Errorf("error %q quotes the body", err)
			}
			if !reflect.DeepEqual(got, domain.TicketDraft{}) {
				t.Errorf("draft = %+v on error, want zero", got)
			}
		})
	}
}
