package security

import (
	"errors"
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
