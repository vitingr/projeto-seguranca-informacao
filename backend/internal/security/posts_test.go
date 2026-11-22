package security

import (
	"errors"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

// TestValidatePost covers 005 S-2: lengths in code points after trimming,
// blank fields, control characters and invalid UTF-8.
func TestValidatePost(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		name      string
		title     string
		body      string
		valid     bool
		wantTitle string
		wantBody  string
	}{
		{"simple", "Olá", "Primeiro post.", true, "Olá", "Primeiro post."},
		{"trimmed", "  Título \t", "\n corpo \n", true, "Título", "corpo"},
		{"title 119", strings.Repeat("a", 119), "b", true, strings.Repeat("a", 119), "b"},
		{"title 120", strings.Repeat("a", 120), "b", true, strings.Repeat("a", 120), "b"},
		{"title 121", strings.Repeat("a", 121), "b", false, "", ""},
		{"title 120 multibyte", strings.Repeat("é", 120), "b", true, strings.Repeat("é", 120), "b"},
		{"title 121 multibyte", strings.Repeat("😀", 121), "b", false, "", ""},
		{"title 120 after trim", " " + strings.Repeat("a", 120) + " ", "b", true, strings.Repeat("a", 120), "b"},
		{"body 2000", "a", strings.Repeat("b", 2000), true, "a", strings.Repeat("b", 2000)},
		{"body 2001", "a", strings.Repeat("b", 2001), false, "", ""},
		{"body 2000 multibyte", "a", strings.Repeat("ç", 2000), true, "a", strings.Repeat("ç", 2000)},
		{"body with newlines and tabs", "a", "linha 1\r\nlinha 2\n\tfim", true, "a", "linha 1\r\nlinha 2\n\tfim"},
		{"empty title", "", "b", false, "", ""},
		{"empty body", "a", "", false, "", ""},
		{"whitespace title", " \t\n ", "b", false, "", ""},
		{"whitespace body", "a", "\n\n  \t", false, "", ""},
		{"newline in title", "a\nb", "b", false, "", ""},
		{"tab in title", "a\tb", "b", false, "", ""},
		{"NUL in title", "a\x00b", "b", false, "", ""},
		{"NUL in body", "a", "b\x00c", false, "", ""},
		{"escape in body", "a", "b\x1bc", false, "", ""},
		{"C1 control in body", "a", "b\u0085c", false, "", ""},
		{"invalid utf-8 title", "a\xffb", "b", false, "", ""},
		{"invalid utf-8 body", "a", "b\xc3", false, "", ""},
		{"html kept as text", "<b>x</b>", `<img src=x onerror="alert(1)">`, true, "<b>x</b>", `<img src=x onerror="alert(1)">`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			title, body, err := s.ValidatePost(tt.title, tt.body)
			if !tt.valid {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("ValidatePost = %v, want ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidatePost = %v, want nil", err)
			}
			if title != tt.wantTitle || body != tt.wantBody {
				t.Errorf("ValidatePost = (%q, %q), want (%q, %q)", title, body, tt.wantTitle, tt.wantBody)
			}
		})
	}
}

func TestParsePage(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		raw  string
		want int // 0: invalid
	}{
		{"", 1},
		{"1", 1},
		{"2", 2},
		{"10000", MaxPostPage},
		{"10001", 0},
		{"99999", 0},
		{"0", 0},
		{"-1", 0},
		{"+1", 0},
		{"01", 0},
		{" 1", 0},
		{"1.5", 0},
		{"abc", 0},
		{"1e3", 0},
		{"999999999999999999999", 0},
	}
	for _, tt := range tests {
		got, err := s.ParsePage(tt.raw)
		if tt.want == 0 {
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("ParsePage(%q) = %d, %v, want ErrInvalidInput", tt.raw, got, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ParsePage(%q) = %d, %v, want %d", tt.raw, got, err, tt.want)
		}
	}
}

func TestParsePostID(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		raw  string
		want domain.PostID // 0: not found
	}{
		{"1", 1},
		{"42", 42},
		{"9223372036854775807", 9223372036854775807},
		{"9223372036854775808", 0}, // overflows int64
		{"0", 0},
		{"-1", 0},
		{"+1", 0},
		{"01", 0},
		{"abc", 0},
		{"", 0},
		{"1 ", 0},
	}
	for _, tt := range tests {
		got, err := s.ParsePostID(tt.raw)
		if tt.want == 0 {
			if !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("ParsePostID(%q) = %d, %v, want ErrNotFound", tt.raw, got, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ParsePostID(%q) = %d, %v, want %d", tt.raw, got, err, tt.want)
		}
	}
}
