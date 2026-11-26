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

// DecodeJSON reads a JSON request body into dst, a pointer to a struct
// whose fields carry `json` tags. Besides the content type, the size limit
// and the trailing data checks, it closes two gaps that encoding/json
// leaves open (plan P-10 and P-11):
//
//   - encoding/json matches keys in any case and keeps the last of duplicate
//     keys, so {"login":"a",…,"LOGIN":"b"} would be read differently by
//     another parser. Every key must be one of the struct's `json` names,
//     spelled exactly, and no key may repeat, even in another case.
//   - encoding/json turns each invalid UTF-8 byte into U+FFFD without an
//     error. A body that is not valid UTF-8 is refused as a whole.
//
// Every error wraps domain.ErrInvalidInput with a fixed reason and never
// quotes the body.
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

// readJSON reads the one JSON value of a request body, with no knowledge of
// its shape: the content type must be application/json (charset utf-8 when
// given), the body at most MaxBodyBytes, nothing may follow the value, and
// the bytes must be valid UTF-8. DecodeJSON and DecodeTicket both start here.
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

// errNotAStruct is a programming error: DecodeJSON only reads into structs,
// whose `json` tags are the list of accepted keys. It is not ErrInvalidInput,
// so the client gets a 500 and the request is refused (fail closed).
var errNotAStruct = errors.New("security: decode json: destination is not a pointer to a struct")

// jsonKeys returns the keys a struct accepts: the name in each exported
// field's `json` tag, or the field name when the tag gives none. Fields
// tagged "-" accept no key. Embedded structs are not supported, so a
// destination with one is refused instead of being read loosely.
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

// NormalizeLogin returns the canonical form of a login (lowercase). It does
// not validate the login: the login flow accepts any value and answers 401
// when it does not authenticate (D-12).
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

// normalizePassword returns the NFKC form of a password, so that visually
// equivalent inputs (such as full-width digits) hash to the same value.
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
