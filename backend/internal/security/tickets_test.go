package security

import (
	"errors"
	"io"
	"strings"
	"testing"

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
