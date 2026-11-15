package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"backend_si/internal/domain"
)

// MaxBodyBytes limita o corpo das requisições JSON aceitas (proteção contra DoS).
const MaxBodyBytes = 64 << 10

// Password policy, counted in Unicode code points of the NFKC form.
const (
	minPasswordLength = 15
	maxPasswordLength = 128
)

// loginPattern is the allowlist for logins of new users.
var loginPattern = regexp.MustCompile(`^[a-z0-9._-]{3,64}$`)

// Reasons for a rejected body. They are fixed strings on purpose: the
// encoding/json errors quote bytes and field names from the body (which may
// hold the password), so they are never wrapped or formatted into the
// returned error.
const (
	reasonContentType = "unsupported content type"
	reasonTooLarge    = "body too large"
	reasonEmpty       = "empty body"
	reasonMalformed   = "malformed json"
	reasonTrailing    = "trailing data"
)

func invalidBody(reason string) error {
	return fmt.Errorf("security: decode json: %s: %w", reason, domain.ErrInvalidInput)
}

// DecodeJSON lê o corpo da requisição em dst com as seguintes regras:
// Content-Type application/json obrigatório, corpo limitado a MaxBodyBytes,
// campos desconhecidos rejeitados e um único objeto JSON por requisição.
// Qualquer violação retorna erro que encapsula domain.ErrInvalidInput.
//
// The charset parameter, when present, must be utf-8. The returned error
// never contains data from the body, so it is safe to log.
func (s *Security) DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return invalidBody(reasonContentType)
	}
	if charset, ok := params["charset"]; ok && !strings.EqualFold(charset, "utf-8") {
		return invalidBody(reasonContentType)
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		switch {
		case isTooLarge(err):
			return invalidBody(reasonTooLarge)
		case errors.Is(err, io.EOF):
			return invalidBody(reasonEmpty)
		default:
			return invalidBody(reasonMalformed)
		}
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if isTooLarge(err) {
			return invalidBody(reasonTooLarge)
		}
		return invalidBody(reasonTrailing)
	}
	return nil
}

func isTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}

// NormalizeLogin returns the canonical form of a login (lowercase). It does
// not validate the login: the login flow accepts any value and answers 401
// when it does not authenticate (D-12).
func (s *Security) NormalizeLogin(raw string) string {
	return strings.ToLower(raw)
}

// ValidateLogin valida o login por lista de permissão (allowlist) de
// caracteres e tamanho.
//
// The rule is ^[a-z0-9._-]{3,64}$, applied to the value as given; callers
// pass it through NormalizeLogin first.
func (s *Security) ValidateLogin(login string) error {
	if !loginPattern.MatchString(login) {
		return fmt.Errorf("security: validate login: %w", domain.ErrInvalidInput)
	}
	return nil
}

// ValidatePassword aplica a política de senha (tamanho mínimo e máximo).
//
// The length is counted in code points of the NFKC form, the same form that
// is hashed (D-14).
func (s *Security) ValidatePassword(password string) error {
	n := utf8.RuneCountInString(normalizePassword(password))
	if n < minPasswordLength || n > maxPasswordLength {
		return fmt.Errorf("security: validate password: %w", domain.ErrInvalidInput)
	}
	return nil
}

// normalizePassword returns the NFKC form of a password, so that visually
// equivalent inputs (such as full-width digits) hash to the same value.
func normalizePassword(password string) string {
	return norm.NFKC.String(password)
}

// ParseRole converte o valor recebido na requisição em um papel válido.
//
// Only the exact values admin and comum are accepted: no case folding and no
// trimming (003 D-7). The error wraps domain.ErrInvalidInput and never quotes
// the input.
func (s *Security) ParseRole(raw string) (domain.Role, error) {
	role := domain.Role(raw)
	if !role.Valid() {
		return "", fmt.Errorf("security: parse role: %w", domain.ErrInvalidInput)
	}
	return role, nil
}
