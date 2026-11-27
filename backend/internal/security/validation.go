package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"backend_si/internal/domain"
)

const MaxBodyBytes = 64 << 10

const (
	minPasswordLength = 15
	maxPasswordLength = 128
)

var loginPattern = regexp.MustCompile(`^[a-z0-9._-]{3,64}$`)

// Fixed strings: encoding/json errors quote the body, which may hold the password.
const (
	reasonContentType  = "unsupported content type"
	reasonTooLarge     = "body too large"
	reasonEmpty        = "empty body"
	reasonMalformed    = "malformed json"
	reasonTrailing     = "trailing data"
	reasonDuplicateKey = "duplicate key"
	reasonInvalidUTF8  = "invalid utf-8"
)

func invalidBody(reason string) error {
	return fmt.Errorf("security: decode json: %s: %w", reason, domain.ErrInvalidInput)
}

func (s *Security) DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	allowed, err := jsonKeys(dst)
	if err != nil {
		return err
	}
	raw, err := readJSON(w, r)
	if err != nil {
		return err
	}
	return decodeObject(raw, allowed, dst)
}

func readJSON(w http.ResponseWriter, r *http.Request) (json.RawMessage, error) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, invalidBody(reasonContentType)
	}
	if charset, ok := params["charset"]; ok && !strings.EqualFold(charset, "utf-8") {
		return nil, invalidBody(reasonContentType)
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	dec := json.NewDecoder(r.Body)

	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		switch {
		case isTooLarge(err):
			return nil, invalidBody(reasonTooLarge)
		case errors.Is(err, io.EOF):
			return nil, invalidBody(reasonEmpty)
		default:
			return nil, invalidBody(reasonMalformed)
		}
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if isTooLarge(err) {
			return nil, invalidBody(reasonTooLarge)
		}
		return nil, invalidBody(reasonTrailing)
	}
	if !utf8.Valid(raw) {
		return nil, invalidBody(reasonInvalidUTF8)
	}
	return raw, nil
}

var errNotAStruct = errors.New("security: decode json: destination is not a pointer to a struct")

func jsonKeys(dst any) ([]string, error) {
	t := reflect.TypeOf(dst)
	if t == nil || t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return nil, errNotAStruct
	}
	t = t.Elem()
	keys := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous {
			return nil, errNotAStruct
		}
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" {
			name = f.Name
		}
		keys = append(keys, name)
	}
	return keys, nil
}

func isTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}

func (s *Security) NormalizeLogin(raw string) string {
	return strings.ToLower(raw)
}

func (s *Security) ValidateLogin(login string) error {
	if !loginPattern.MatchString(login) {
		return fmt.Errorf("security: validate login: %w", domain.ErrInvalidInput)
	}
	return nil
}

func (s *Security) ValidatePassword(password string) error {
	n := utf8.RuneCountInString(normalizePassword(password))
	if n < minPasswordLength || n > maxPasswordLength {
		return fmt.Errorf("security: validate password: %w", domain.ErrInvalidInput)
	}
	return nil
}

// NFKC makes visually equivalent inputs hash to the same value.
func normalizePassword(password string) string {
	return norm.NFKC.String(password)
}

func (s *Security) ParseRole(raw string) (domain.Role, error) {
	role := domain.Role(raw)
	if !role.Valid() {
		return "", fmt.Errorf("security: parse role: %w", domain.ErrInvalidInput)
	}
	return role, nil
}
