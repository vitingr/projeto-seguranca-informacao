package security

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

const messageMarker = "Texto-Secreto-41b9"

func decodeMessage(t *testing.T, s *Security, contentType, body string) (string, error) {
	t.Helper()
	return s.DecodeMessage(httptest.NewRecorder(), newJSONRequest(t, contentType, body))
}

func TestDecodeMessageAccepts(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	tests := []struct {
		name string
		body string
		want string
	}{
		{"plain", `{"texto":"Olá, ainda está acontecendo."}`, "Olá, ainda está acontecendo."},
		{"kept raw (not trimmed yet)", `{"texto":"  Oi\n"}`, "  Oi\n"},
		{"escapes", `{"texto":"linha 1\nlinha 2\té 😀"}`, "linha 1\nlinha 2\té 😀"},
		{"markup kept as text", `{"texto":"<script>alert(1)</script>"}`, "<script>alert(1)</script>"},
		{"empty string decodes (ValidateMessage refuses it)", `{"texto":""}`, ""},
		{"spaces around the JSON", " \n{ \"texto\" : \"Oi\" }\n ", "Oi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeMessage(t, s, "application/json", tt.body)
			if err != nil {
				t.Fatalf("DecodeMessage(%q): %v", tt.body, err)
			}
			if got != tt.want {
				t.Errorf("DecodeMessage(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
	if got, err := decodeMessage(t, s, "application/json; charset=utf-8", `{"texto":"Oi"}`); err != nil || got != "Oi" {
		t.Errorf("charset=utf-8: %q, %v; want \"Oi\", nil", got, err)
	}
}

func TestDecodeMessageRefuses(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	const ct = "application/json"
	text := `"texto":"` + messageMarker + `"`
	invalidByte := string([]byte{0xff})
	tests := []struct {
		name        string
		contentType string
		body        string
		reason      string
	}{
		{"author", ct, `{` + text + `,"author":1}`, reasonMalformed},
		{"user_id", ct, `{` + text + `,"user_id":1}`, reasonMalformed},
		{"created_at", ct, `{` + text + `,"created_at":"2020-01-01T00:00:00Z"}`, reasonMalformed},
		{"id", ct, `{` + text + `,"id":1}`, reasonMalformed},
		{"ticket_id", ct, `{` + text + `,"ticket_id":2}`, reasonMalformed},
		{"status", ct, `{` + text + `,"status":"fechado"}`, reasonMalformed},
		{"kind", ct, `{` + text + `,"kind":"status_change"}`, reasonMalformed},
		{"text in english", ct, `{"text":"` + messageMarker + `"}`, reasonMalformed},

		{"Texto", ct, `{"Texto":"` + messageMarker + `"}`, reasonMalformed},
		{"TEXTO", ct, `{"TEXTO":"` + messageMarker + `"}`, reasonMalformed},
		{"duplicate texto", ct, `{` + text + `,"texto":"outro"}`, reasonDuplicateKey},
		{"duplicate in another case", ct, `{` + text + `,"TEXTO":"outro"}`, reasonDuplicateKey},

		{"empty object", ct, `{}`, reasonNoText},
		{"texto null", ct, `{"texto":null}`, reasonNoText},
		{"texto number", ct, `{"texto":1}`, reasonMalformed},
		{"texto bool", ct, `{"texto":true}`, reasonMalformed},
		{"texto array", ct, `{"texto":["` + messageMarker + `"]}`, reasonMalformed},
		{"texto object", ct, `{"texto":{"x":"` + messageMarker + `"}}`, reasonMalformed},
		{"body null", ct, `null`, reasonMalformed},
		{"body string", ct, `"` + messageMarker + `"`, reasonMalformed},
		{"body array", ct, `[{` + text + `}]`, reasonMalformed},

		{"trailing data", ct, `{` + text + `}{"x":1}`, reasonTrailing},
		{"invalid utf-8 in the text", ct, `{"texto":"Ol` + invalidByte + `á ` + messageMarker + `"}`, reasonInvalidUTF8},
		{"invalid utf-8 in a key", ct, `{` + text + `,"` + invalidByte + `":1}`, reasonInvalidUTF8},
		{"wrong content type", "text/plain", `{` + text + `}`, reasonContentType},
		{"form content type", "application/x-www-form-urlencoded", `texto=` + messageMarker, reasonContentType},
		{"no content type", "", `{` + text + `}`, reasonContentType},
		{"other charset", "application/json; charset=latin1", `{` + text + `}`, reasonContentType},
		{"empty body", ct, ``, reasonEmpty},
		{"body over 64 KiB", ct, `{"texto":"` + rep(messageMarker, MaxBodyBytes/len(messageMarker)+1) + `"}`, reasonTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeMessage(t, s, tt.contentType, tt.body)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput (400)", err)
			}
			if err.Error() != invalidBody(tt.reason).Error() {
				t.Errorf("error = %q, want the fixed %q", err, invalidBody(tt.reason))
			}
			if strings.Contains(err.Error(), messageMarker) {
				t.Errorf("error %q quotes the body", err)
			}
			if got != "" {
				t.Errorf("text = %q on error, want empty", got)
			}
		})
	}
}

func TestDecodeMessageLargestValid(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	body := `{"texto":"` + rep(`😀`, MessageMaxLength) + `"}`
	raw, err := decodeMessage(t, s, "application/json", body)
	if err != nil {
		t.Fatalf("DecodeMessage(%d bytes): %v", len(body), err)
	}
	if got, err := s.ValidateMessage(raw); err != nil || got != rep("😀", MessageMaxLength) {
		t.Errorf("ValidateMessage = %d bytes, %v; want %d emoji", len(got), err, MessageMaxLength)
	}
}

func TestValidateMessage(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	runFieldCases(t, s.ValidateMessage, []fieldCase{
		{"plain", "Olá, ainda está acontecendo.", "Olá, ainda está acontecendo.", ""},
		{"trimmed", "\n\n  Obrigado!  \t\n", "Obrigado!", ""},
		{"line breaks and tab kept inside", "1. Abrir\r\n2. Entrar\n\tdepois", "1. Abrir\r\n2. Entrar\n\tdepois", ""},
		{"markup kept as typed", "<b>oi</b> **x** javascript:alert(1)", "<b>oi</b> **x** javascript:alert(1)", ""},
		{"line separator allowed", "linha um linha dois", "linha um linha dois", ""},
		{"paragraph separator allowed", "um dois", "um dois", ""},

		{"empty", "", "", fieldEmpty},
		{"spaces only", "     ", "", fieldEmpty},
		{"line breaks only", "\n\r\n\n", "", fieldEmpty},
		{"tabs and breaks only", "\t \n\t", "", fieldEmpty},

		{"1 character", "a", "a", ""},
		{"1 multi-byte character", "😀", "😀", ""},
		{"max", rep("a", MessageMaxLength), rep("a", MessageMaxLength), ""},
		{"max + 1", rep("a", MessageMaxLength+1), "", fieldTooLong},
		{"multi-byte at the max", rep("ç", MessageMaxLength-1) + "😀", rep("ç", MessageMaxLength-1) + "😀", ""},
		{"multi-byte past the max", rep("ç", MessageMaxLength) + "😀", "", fieldTooLong},
		{"max after trimming", "  " + rep("é", MessageMaxLength) + "\n", rep("é", MessageMaxLength), ""},

		{"NUL", "Oi\x00 tudo bem", "", fieldControl},
		{"escape", "Oi\x1b[31m tudo bem", "", fieldControl},
		{"vertical tab", "Oi\v tudo bem", "", fieldControl},
		{"form feed", "Oi\f tudo bem", "", fieldControl},
		{"backspace", "Oi\b tudo bem", "", fieldControl},
		{"DEL", "Oi\x7f tudo bem", "", fieldControl},
		{"C1 control", "Oi\u0085 tudo bem", "", fieldControl},
		{"invalid utf-8", "Oi \xff tudo bem", "", fieldInvalidUTF8},
	})
}

func TestValidateMessageErrorsNeverQuoteTheText(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, io.Discard)
	for _, raw := range []string{
		messageMarker + "\x00",
		messageMarker + rep("a", MessageMaxLength),
		messageMarker + "\xff",
	} {
		_, err := s.ValidateMessage(raw)
		if err == nil {
			t.Fatalf("ValidateMessage accepted %q", raw)
		}
		if strings.Contains(err.Error(), messageMarker) {
			t.Errorf("error %q quotes the text", err)
		}
	}
}
