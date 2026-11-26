package security

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"backend_si/internal/domain"
)

// Ticket field rules (spec 001-ticket-builder, section 4). Every rule takes
// the raw value, trims leading and trailing whitespace, and returns the
// trimmed value or an error wrapping domain.ErrInvalidInput. Lengths are
// Unicode code points. The errors carry a fixed reason and never the value:
// ticket fields may hold sensitive data (S-7).

// fieldRule validates one raw field value and returns its stored form.
type fieldRule func(raw string) (string, error)

const (
	fieldInvalidUTF8 = "invalid utf-8"
	fieldEmpty       = "empty"
	fieldTooShort    = "too short"
	fieldTooLong     = "too long"
	fieldControl     = "control character"
	fieldLineBreak   = "line break in a single-line field"
	fieldNotHTTPSURL = "not an https url"
	fieldCredentials = "credentials in url"
)

func invalidField(reason string) error {
	return fmt.Errorf("security: validate ticket field: %s: %w", reason, domain.ErrInvalidInput)
}

// singleLine accepts text with no control character at all (no line break,
// tab or NUL) and no Unicode line or paragraph separator, between min and
// max code points after trimming. min = 0 makes the field optional.
func singleLine(minLen, maxLen int) fieldRule {
	return func(raw string) (string, error) {
		return text(raw, minLen, maxLen, func(r rune) error {
			switch {
			case r == '\n' || r == '\r' || r == ' ' || r == ' ':
				return invalidField(fieldLineBreak)
			case unicode.IsControl(r):
				return invalidField(fieldControl)
			}
			return nil
		})
	}
}

// multiLine accepts text whose only control characters are line feed,
// carriage return and tab, between min and max code points after trimming.
// min = 0 makes the field optional.
func multiLine(minLen, maxLen int) fieldRule {
	return func(raw string) (string, error) {
		return text(raw, minLen, maxLen, func(r rune) error {
			if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
				return invalidField(fieldControl)
			}
			return nil
		})
	}
}

// httpsURL accepts an absolute https URL with a host and no user name or
// password, up to max code points, with no whitespace or control character
// inside. SSH addresses (git@host:repo) and any other scheme are refused.
func httpsURL(maxLen int) fieldRule {
	single := singleLine(1, maxLen)
	return func(raw string) (string, error) {
		value, err := single(raw)
		if err != nil {
			return "", err
		}
		if strings.IndexFunc(value, unicode.IsSpace) >= 0 {
			return "", invalidField(fieldNotHTTPSURL)
		}
		u, err := url.Parse(value)
		if err != nil || u.Scheme != "https" || u.Opaque != "" || u.Hostname() == "" {
			return "", invalidField(fieldNotHTTPSURL)
		}
		if u.User != nil {
			return "", invalidField(fieldCredentials)
		}
		return value, nil
	}
}

// text applies the rules every text field shares: valid UTF-8, trimming,
// the per-character check, then the length in code points.
func text(raw string, minLen, maxLen int, check func(rune) error) (string, error) {
	if !utf8.ValidString(raw) {
		return "", invalidField(fieldInvalidUTF8)
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		if minLen > 0 {
			return "", invalidField(fieldEmpty)
		}
		return "", nil
	}
	for _, r := range value {
		if err := check(r); err != nil {
			return "", err
		}
	}
	switch n := utf8.RuneCountInString(value); {
	case n < minLen:
		return "", invalidField(fieldTooShort)
	case n > maxLen:
		return "", invalidField(fieldTooLong)
	}
	return value, nil
}
