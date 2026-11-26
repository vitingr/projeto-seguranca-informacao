package security

import (
	"fmt"
	"net/url"
	"slices"
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

// invalidField is the error of a field rule: only the reason, since the
// caller wraps it with the name of the field.
func invalidField(reason string) error {
	return fmt.Errorf("%s: %w", reason, domain.ErrInvalidInput)
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
//
//nolint:unparam // the maximum is part of the rule, as in singleLine; spec section 4 happens to use one value.
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

// Limits of spec section 4.
const (
	ticketShortMin = 2
	ticketShortMax = 120
	ticketLongMin  = 10
	ticketLongMax  = 2000
	ticketURLMax   = 300
)

var (
	shortText    = singleLine(ticketShortMin, ticketShortMax)
	longText     = multiLine(ticketLongMin, ticketLongMax)
	optionalText = multiLine(0, ticketLongMax)
	repoURL      = httpsURL(ticketURLMax)
)

// ticketFields names the draft fields each type has, by their request key.
var ticketFields = map[domain.TicketType][]string{
	domain.TicketTypeAccess: {"sistema", "nivel_acesso", "justificativa"},
	domain.TicketTypeBug:    {"sistema", "descricao", "passos", "severidade"},
	domain.TicketTypeDeploy: {"projeto", "repositorio", "ambiente", "observacoes"},
	domain.TicketTypeOther:  {"titulo", "descricao"},
}

// ValidateTicket applies the rules of spec section 4 to a decoded request
// and returns the ticket with trimmed values (BR-4). The type must be known,
// every field of the type valid, and no field of another type set. Empty
// optional notes become "". The title is set only for "outro"; the service
// builds the others (BR-3). Author, status and dates are left for the
// service and the database. Errors wrap domain.ErrInvalidInput and name the
// field, never its value.
func (s *Security) ValidateTicket(d domain.TicketDraft) (domain.Ticket, error) {
	allowed, ok := ticketFields[d.Type]
	if !ok {
		return domain.Ticket{}, fmt.Errorf("security: validate ticket: unknown type: %w", domain.ErrInvalidInput)
	}
	for name, set := range draftFieldsSet(d) {
		if set && !slices.Contains(allowed, name) {
			return domain.Ticket{}, fmt.Errorf("security: validate ticket: %s: not a field of the type: %w", name, domain.ErrInvalidInput)
		}
	}

	v := ticketValidator{}
	t := domain.Ticket{Type: d.Type}
	switch d.Type {
	case domain.TicketTypeAccess:
		t.System = v.text("sistema", shortText, d.System)
		t.AccessLevel = domain.AccessLevel(v.choice("nivel_acesso", d.AccessLevel, domain.AccessLevel(d.AccessLevel).Valid()))
		t.Justification = v.text("justificativa", longText, d.Justification)
	case domain.TicketTypeBug:
		t.System = v.text("sistema", shortText, d.System)
		t.Description = v.text("descricao", longText, d.Description)
		t.Steps = v.text("passos", longText, d.Steps)
		t.Severity = domain.Severity(v.choice("severidade", d.Severity, domain.Severity(d.Severity).Valid()))
	case domain.TicketTypeDeploy:
		t.Project = v.text("projeto", shortText, d.Project)
		t.Repository = v.text("repositorio", repoURL, d.Repository)
		t.Environment = domain.Environment(v.choice("ambiente", d.Environment, domain.Environment(d.Environment).Valid()))
		if d.Notes != nil {
			t.Notes = v.text("observacoes", optionalText, *d.Notes)
		}
	case domain.TicketTypeOther:
		t.Title = v.text("titulo", shortText, d.Title)
		t.Description = v.text("descricao", longText, d.Description)
	}
	if v.err != nil {
		return domain.Ticket{}, v.err
	}
	return t, nil
}

// draftFieldsSet tells, by request key, which draft fields hold a value.
// Notes count as set when present, even if empty.
func draftFieldsSet(d domain.TicketDraft) map[string]bool {
	return map[string]bool{
		"titulo":        d.Title != "",
		"sistema":       d.System != "",
		"nivel_acesso":  d.AccessLevel != "",
		"justificativa": d.Justification != "",
		"descricao":     d.Description != "",
		"passos":        d.Steps != "",
		"severidade":    d.Severity != "",
		"projeto":       d.Project != "",
		"repositorio":   d.Repository != "",
		"ambiente":      d.Environment != "",
		"observacoes":   d.Notes != nil,
	}
}

// ticketValidator keeps the first error, so ValidateTicket reads as a list
// of fields.
type ticketValidator struct{ err error }

func (v *ticketValidator) text(name string, rule fieldRule, raw string) string {
	if v.err != nil {
		return ""
	}
	value, err := rule(raw)
	if err != nil {
		v.err = fmt.Errorf("security: validate ticket: %s: %w", name, err)
	}
	return value
}

// choice accepts a fixed-choice code exactly as listed: no trimming, no
// case folding (spec section 4).
func (v *ticketValidator) choice(name, raw string, valid bool) string {
	if v.err != nil {
		return ""
	}
	if !valid {
		v.err = fmt.Errorf("security: validate ticket: %s: not one of the listed codes: %w", name, domain.ErrInvalidInput)
		return ""
	}
	return raw
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
