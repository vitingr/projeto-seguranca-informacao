package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

// sentinelPassword marks the password in request bodies, so tests can check
// that it never leaks into errors or logs.
const sentinelPassword = "Sentinel-Pa55word-7f3a"

type loginBody struct {
	Login string `json:"login"`
	Senha string `json:"senha"`
}

func newJSONRequest(t *testing.T, contentType, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/login", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req
}

// padTo returns body followed by spaces up to n bytes.
func padTo(body string, n int) string {
	return body + strings.Repeat(" ", n-len(body))
}

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	const ct = "application/json"
	valid := `{"login":"admin","senha":"` + sentinelPassword + `"}`

	tests := []struct {
		name        string
		contentType string
		body        string
		wantErr     string // reason in the error; empty = success
	}{
		{"valid", ct, valid, ""},
		{"charset utf-8", "application/json; charset=utf-8", valid, ""},
		{"charset UTF-8 upper case", "Application/JSON; charset=UTF-8", valid, ""},
		{"trailing whitespace", ct, valid + " \n\t", ""},
		{"exactly the limit", ct, padTo(valid, MaxBodyBytes), ""},
		{"missing content type", "", valid, reasonContentType},
		{"text/plain", "text/plain", valid, reasonContentType},
		{"json suffix type", "application/problem+json", valid, reasonContentType},
		{"other charset", "application/json; charset=latin1", valid, reasonContentType},
		{"malformed content type", "application/json; charset", valid, reasonContentType},
		{"one byte over the limit", ct, padTo(valid, MaxBodyBytes+1), reasonTooLarge},
		{"huge password", ct, `{"login":"a","senha":"` + strings.Repeat(sentinelPassword, 4000) + `"}`, reasonTooLarge},
		{"empty body", ct, "", reasonEmpty},
		{"whitespace only", ct, "  \n", reasonEmpty},
		{"unknown field", ct, `{"login":"a","senha":"` + sentinelPassword + `","role_` + sentinelPassword + `":1}`, reasonMalformed},
		{"invalid json", ct, `{"login":"a","senha":"` + sentinelPassword + `"`, reasonMalformed},
		{"control byte in password", ct, `{"login":"a","senha":"` + sentinelPassword + "\x01" + `"}`, reasonMalformed},
		{"wrong type", ct, `{"login":"a","senha":["` + sentinelPassword + `"]}`, reasonMalformed},
		{"not an object", ct, `"` + sentinelPassword + `"`, reasonMalformed},
		{"second object", ct, valid + `{"login":"b"}`, reasonTrailing},
		{"trailing garbage", ct, valid + sentinelPassword, reasonTrailing},
		{"trailing data over the limit", ct, padTo(valid+"x", MaxBodyBytes+1), reasonTrailing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestSecurity(t, nil)
			var dst loginBody
			err := s.DecodeJSON(httptest.NewRecorder(), newJSONRequest(t, tt.contentType, tt.body), &dst)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("DecodeJSON() error = %v", err)
				}
				if dst.Login != "admin" || dst.Senha != sentinelPassword {
					t.Errorf("decoded %+v", dst)
				}
				return
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("DecodeJSON() error = %v, want ErrInvalidInput", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("DecodeJSON() error = %q, want reason %q", err, tt.wantErr)
			}
		})
	}
}

// TestDecodeJSONErrorsNeverQuoteTheBody guards the logs: an error from
// DecodeJSON may be logged, so it must be one of the fixed messages and
// never carry bytes or field names from the body.
func TestDecodeJSONErrorsNeverQuoteTheBody(t *testing.T) {
	t.Parallel()

	fixed := map[string]bool{}
	for _, reason := range []string{reasonContentType, reasonTooLarge, reasonEmpty, reasonMalformed, reasonTrailing} {
		fixed[invalidBody(reason).Error()] = true
	}

	bodies := []struct {
		contentType string
		body        string
	}{
		{"text/plain; charset=" + sentinelPassword, sentinelPassword},
		{"application/json", `{"senha":"` + sentinelPassword + "\x01" + `"}`},
		{"application/json", `{"senha":"` + sentinelPassword + `","` + sentinelPassword + `":1}`},
		{"application/json", `{"senha":` + sentinelPassword + `}`},
		{"application/json", `{"senha":{"` + sentinelPassword + `":1}}`},
		{"application/json", `{"senha":"x"}` + sentinelPassword},
		{"application/json", `{"senha":"` + strings.Repeat(sentinelPassword, 4000) + `"}`},
	}
	for _, b := range bodies {
		s := newTestSecurity(t, nil)
		var dst loginBody
		err := s.DecodeJSON(httptest.NewRecorder(), newJSONRequest(t, b.contentType, b.body), &dst)
		if err == nil {
			t.Fatalf("body %q was accepted", b.body)
		}
		msg := err.Error()
		if !fixed[msg] {
			t.Errorf("error %q is not one of the fixed messages", msg)
		}
		if strings.Contains(msg, sentinelPassword) || strings.Contains(msg, `\x01`) {
			t.Errorf("error %q quotes the body", msg)
		}
	}
}

func TestNormalizeLogin(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct{ in, want string }{
		{"admin", "admin"},
		{"Admin.Root", "admin.root"},
		{"ADMIN_1-x", "admin_1-x"},
		{" Padded ", " padded "}, // lowercase only, no trimming (D-12)
		{"", ""},
	}
	for _, tt := range tests {
		if got := s.NormalizeLogin(tt.in); got != tt.want {
			t.Errorf("NormalizeLogin(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateLogin(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		login string
		valid bool
	}{
		{"abc", true},
		{"admin", true},
		{"joao.silva_2-x", true},
		{strings.Repeat("a", 64), true},
		{"ab", false},
		{strings.Repeat("a", 65), false},
		{"", false},
		{"Admin", false}, // callers normalize first
		{"jo\u00e3o", false},
		{"a b c", false},
		{"admin\n", false},
		{"admin@x", false},
		{"../etc", false},
	}
	for _, tt := range tests {
		err := s.ValidateLogin(tt.login)
		if tt.valid && err != nil {
			t.Errorf("ValidateLogin(%q) = %v, want nil", tt.login, err)
		}
		if !tt.valid && !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("ValidateLogin(%q) = %v, want ErrInvalidInput", tt.login, err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"minimum", strings.Repeat("a", 15), true},
		{"maximum", strings.Repeat("a", 128), true},
		{"too short", strings.Repeat("a", 14), false},
		{"too long", strings.Repeat("a", 129), false},
		{"empty", "", false},
		// 15 code points but 30 bytes: length counts code points.
		{"multi-byte minimum", strings.Repeat("\u00e9", 15), true},
		{"multi-byte too long", strings.Repeat("\u00e9", 129), false},
		// "e" + combining acute (2 code points) composes to "\u00e9" (1) under NFKC.
		{"composes below minimum", strings.Repeat("e\u0301", 14), false},
		{"composes to the maximum", strings.Repeat("e\u0301", 128), true},
		// The ligature "\ufb03" (1 code point) expands to "ffi" (3) under NFKC.
		{"expands to the minimum", strings.Repeat("\ufb03", 5), true},
		{"expands above maximum", strings.Repeat("\ufb03", 43), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := s.ValidatePassword(tt.password)
			if tt.valid && err != nil {
				t.Errorf("ValidatePassword() = %v, want nil", err)
			}
			if !tt.valid && !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("ValidatePassword() = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestNormalizePasswordIsNFKC(t *testing.T) {
	t.Parallel()

	tests := []struct{ in, want string }{
		{"e\u0301", "\u00e9"},  // composition
		{"\uff21\uff11", "A1"}, // full-width letters and digits
		{"\ufb03", "ffi"},      // ligature
		{"plain ascii", "plain ascii"},
	}
	for _, tt := range tests {
		if got := normalizePassword(tt.in); got != tt.want {
			t.Errorf("normalizePassword(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseRole(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	tests := []struct {
		raw     string
		want    domain.Role
		wantErr bool
	}{
		{"admin", domain.RoleAdmin, false},
		{"comum", domain.RoleComum, false},
		{"", "", true},
		{"Admin", "", true},
		{"ADMIN", "", true},
		{"Comum", "", true},
		{" admin", "", true},
		{"admin ", "", true},
		{"root", "", true},
		{"comum\x00", "", true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.raw), func(t *testing.T) {
			t.Parallel()
			got, err := s.ParseRole(tt.raw)
			if !tt.wantErr {
				if err != nil || got != tt.want {
					t.Fatalf("ParseRole(%q) = %q, %v; want %q", tt.raw, got, err, tt.want)
				}
				return
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("ParseRole(%q) error = %v, want ErrInvalidInput", tt.raw, err)
			}
			if got != "" {
				t.Errorf("ParseRole(%q) role = %q, want empty", tt.raw, got)
			}
			if tt.raw != "" && strings.Contains(err.Error(), tt.raw) {
				t.Errorf("error %q quotes the input", err)
			}
		})
	}
}

// registrationBody mirrors the registration request (login, senha, papel).
type registrationBody struct {
	Login string `json:"login"`
	Senha string `json:"senha"`
	Papel string `json:"papel"`
}

// decodeInto runs DecodeJSON on body with the JSON content type.
func decodeInto(t *testing.T, s *Security, body string, dst any) error {
	t.Helper()
	return s.DecodeJSON(httptest.NewRecorder(), newJSONRequest(t, "application/json", body), dst)
}

// checkFixedRefusal checks that err is the fixed error of reason: it wraps
// ErrInvalidInput (400) and quotes nothing from the body.
func checkFixedRefusal(t *testing.T, err error, reason string) {
	t.Helper()
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput (400)", err)
	}
	if err.Error() != invalidBody(reason).Error() {
		t.Errorf("error = %q, want the fixed %q", err, invalidBody(reason))
	}
	if strings.Contains(err.Error(), sentinelPassword) {
		t.Errorf("error %q quotes the body", err)
	}
}

// TestDecodeJSONKeySpelling covers plan P-11 for login and registration:
// encoding/json alone matches keys in any case and keeps the last of
// duplicate keys, so {"login":"a",…,"LOGIN":"b"} would be read as "b" here
// and maybe as "a" by a proxy or a log. Every key must be one of the
// struct's json names, spelled exactly, and none may repeat in any case.
func TestDecodeJSONKeySpelling(t *testing.T) {
	t.Parallel()

	const pw = `"senha":"` + sentinelPassword + `"`
	login := []struct {
		name   string
		body   string
		reason string
	}{
		{"LOGIN", `{"LOGIN":"admin",` + pw + `}`, reasonMalformed},
		{"Login", `{"Login":"admin",` + pw + `}`, reasonMalformed},
		{"Senha", `{"login":"admin","Senha":"` + sentinelPassword + `"}`, reasonMalformed},
		{"SENHA", `{"login":"admin","SENHA":"` + sentinelPassword + `"}`, reasonMalformed},
		{"login twice", `{"login":"nobody",` + pw + `,"login":"admin"}`, reasonDuplicateKey},
		{"login and LOGIN", `{"login":"nobody",` + pw + `,"LOGIN":"admin"}`, reasonDuplicateKey},
		{"senha and Senha", `{"login":"admin","senha":"x","Senha":"` + sentinelPassword + `"}`, reasonDuplicateKey},
		// Refused at its first key already, for the spelling.
		{"LOGIN before login", `{"LOGIN":"admin","login":"nobody",` + pw + `}`, reasonMalformed},
	}
	for _, tt := range login {
		t.Run("login/"+tt.name, func(t *testing.T) {
			t.Parallel()
			var dst loginBody
			err := decodeInto(t, newTestSecurity(t, nil), tt.body, &dst)
			checkFixedRefusal(t, err, tt.reason)
		})
	}

	registration := []struct {
		name   string
		body   string
		reason string
	}{
		{"Papel", `{"login":"maria",` + pw + `,"Papel":"admin"}`, reasonMalformed},
		{"PAPEL", `{"login":"maria",` + pw + `,"PAPEL":"admin"}`, reasonMalformed},
		{"LOGIN", `{"LOGIN":"maria",` + pw + `,"papel":"comum"}`, reasonMalformed},
		{"Senha", `{"login":"maria","Senha":"` + sentinelPassword + `","papel":"comum"}`, reasonMalformed},
		{"papel twice", `{"login":"maria",` + pw + `,"papel":"comum","papel":"admin"}`, reasonDuplicateKey},
		{"papel and PAPEL", `{"login":"maria",` + pw + `,"papel":"comum","PAPEL":"admin"}`, reasonDuplicateKey},
		{"login and Login", `{"login":"maria",` + pw + `,"papel":"comum","Login":"admin"}`, reasonDuplicateKey},
	}
	for _, tt := range registration {
		t.Run("registration/"+tt.name, func(t *testing.T) {
			t.Parallel()
			var dst registrationBody
			err := decodeInto(t, newTestSecurity(t, nil), tt.body, &dst)
			checkFixedRefusal(t, err, tt.reason)
		})
	}

	// The documented keys, exactly spelled, still decode.
	var dst registrationBody
	if err := decodeInto(t, newTestSecurity(t, nil), `{"login":"maria",`+pw+`,"papel":"comum"}`, &dst); err != nil {
		t.Fatalf("exact keys: %v", err)
	}
	if dst != (registrationBody{Login: "maria", Senha: sentinelPassword, Papel: "comum"}) {
		t.Errorf("decoded %+v", dst)
	}
}

// TestDecodeJSONRefusesInvalidUTF8 covers plan P-11 at the body level:
// encoding/json alone would turn each invalid byte into U+FFFD and accept
// the login or the password.
func TestDecodeJSONRefusesInvalidUTF8(t *testing.T) {
	t.Parallel()

	// Built from bytes, so no tool can rewrite them as valid characters.
	var (
		invalidByte = string([]byte{0xff})
		truncated   = string([]byte{0xc3})
		overlong    = string([]byte{0xc0, 0xaf})
		surrogate   = string([]byte{0xed, 0xa0, 0x80})
	)
	tests := []struct {
		name string
		body string
	}{
		{"invalid byte in login", `{"login":"adm` + invalidByte + `in","senha":"` + sentinelPassword + `"}`},
		{"invalid byte in senha", `{"login":"admin","senha":"` + sentinelPassword + invalidByte + `"}`},
		{"truncated character at the end of senha", `{"login":"admin","senha":"` + sentinelPassword + truncated + `"}`},
		{"overlong encoding", `{"login":"admin` + overlong + `","senha":"` + sentinelPassword + `"}`},
		{"utf-16 surrogate encoded as utf-8", `{"login":"admin","senha":"` + sentinelPassword + surrogate + `"}`},
		{"invalid byte in a key", `{"login":"admin","senha":"` + sentinelPassword + `","` + invalidByte + `":"x"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dst loginBody
			err := decodeInto(t, newTestSecurity(t, nil), tt.body, &dst)
			checkFixedRefusal(t, err, reasonInvalidUTF8)
			if dst != (loginBody{}) {
				t.Errorf("decoded %+v on error, want zero", dst)
			}
		})
	}

	// The same kinds of characters, written correctly, decode as typed.
	var dst loginBody
	if err := decodeInto(t, newTestSecurity(t, nil), `{"login":"joão","senha":"Sênha-çã-😀-`+sentinelPassword+`"}`, &dst); err != nil {
		t.Fatalf("valid UTF-8 body: %v", err)
	}
	if dst.Login != "joão" || dst.Senha != "Sênha-çã-😀-"+sentinelPassword {
		t.Errorf("decoded %+v", dst)
	}
}

// TestDecodeJSONRefusesBodiesThatAreNotObjects: the key rule needs an
// object, so a JSON null (which encoding/json reads into a struct as "no
// fields") or an array is refused like any other wrong shape.
func TestDecodeJSONRefusesBodiesThatAreNotObjects(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`null`, `[]`, `["admin"]`, `1`, `true`} {
		var dst loginBody
		err := decodeInto(t, newTestSecurity(t, nil), body, &dst)
		checkFixedRefusal(t, err, reasonMalformed)
	}
	var dst loginBody
	if err := decodeInto(t, newTestSecurity(t, nil), `{}`, &dst); err != nil {
		t.Errorf("empty object: %v (missing keys are left to the caller's rules)", err)
	}
}

// TestDecodeJSONKeysComeFromTheTags checks how the accepted keys are read
// from the destination, and that a destination the rule cannot describe is
// refused as a server error (not ErrInvalidInput), so it fails closed.
func TestDecodeJSONKeysComeFromTheTags(t *testing.T) {
	t.Parallel()

	type tagged struct {
		Named    string `json:"named,omitempty"`
		Untagged string
		Skipped  string `json:"-"`
		hidden   string //nolint:unused // proves unexported fields accept no key
	}
	keys, err := jsonKeys(&tagged{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"named", "Untagged"}; !slices.Equal(keys, want) {
		t.Errorf("keys = %q, want %q", keys, want)
	}

	s := newTestSecurity(t, nil)
	var got tagged
	if err := decodeInto(t, s, `{"named":"a","Untagged":"b"}`, &got); err != nil {
		t.Fatalf("exact keys: %v", err)
	}
	if got.Named != "a" || got.Untagged != "b" {
		t.Errorf("decoded %+v", got)
	}
	for _, body := range []string{`{"Skipped":"x"}`, `{"hidden":"x"}`, `{"untagged":"x"}`, `{"Named":"x"}`} {
		var dst tagged
		checkFixedRefusal(t, decodeInto(t, s, body, &dst), reasonMalformed)
	}

	type embedded struct{ loginBody }
	var raw json.RawMessage
	var m map[string]string
	for name, dst := range map[string]any{
		"nil":                nil,
		"struct value":       loginBody{},
		"raw message":        &raw,
		"map":                &m,
		"embedded struct":    &embedded{},
		"pointer to pointer": new(*loginBody),
	} {
		err := decodeInto(t, s, `{"login":"admin"}`, dst)
		if !errors.Is(err, errNotAStruct) || errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want errNotAStruct and not ErrInvalidInput", name, err)
		}
	}
}
