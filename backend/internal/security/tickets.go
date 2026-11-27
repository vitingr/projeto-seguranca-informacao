package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"backend_si/internal/domain"
)

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
	return fmt.Errorf("%s: %w", reason, domain.ErrInvalidInput)
}

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

//nolint:unparam // the maximum is part of the rule, as in singleLine
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

type ticketRequest struct {
	Tipo  string          `json:"tipo"`
	Dados json.RawMessage `json:"dados"`
}

type accessData struct {
	Sistema       string `json:"sistema"`
	NivelAcesso   string `json:"nivel_acesso"`
	Justificativa string `json:"justificativa"`
}

type bugData struct {
	Sistema    string `json:"sistema"`
	Descricao  string `json:"descricao"`
	Passos     string `json:"passos"`
	Severidade string `json:"severidade"`
}

type deployData struct {
	Projeto     string  `json:"projeto"`
	Repositorio string  `json:"repositorio"`
	Ambiente    string  `json:"ambiente"`
	Observacoes *string `json:"observacoes"`
}

type otherData struct {
	Titulo    string `json:"titulo"`
	Descricao string `json:"descricao"`
}

const reasonNoData = "missing dados"

var ticketRequestKeys = []string{"tipo", "dados"}

func (s *Security) DecodeTicket(w http.ResponseWriter, r *http.Request) (domain.TicketDraft, error) {
	raw, err := readJSON(w, r)
	if err != nil {
		return domain.TicketDraft{}, err
	}
	var req ticketRequest
	if err := decodeObject(raw, ticketRequestKeys, &req); err != nil {
		return domain.TicketDraft{}, err
	}
	if len(req.Dados) == 0 || bytes.Equal(req.Dados, []byte("null")) {
		return domain.TicketDraft{}, invalidBody(reasonNoData)
	}

	d := domain.TicketDraft{Type: domain.TicketType(req.Tipo)}
	keys := ticketFields[d.Type]
	switch d.Type {
	case domain.TicketTypeAccess:
		var v accessData
		if err := decodeObject(req.Dados, keys, &v); err != nil {
			return domain.TicketDraft{}, err
		}
		d.System, d.AccessLevel, d.Justification = v.Sistema, v.NivelAcesso, v.Justificativa
	case domain.TicketTypeBug:
		var v bugData
		if err := decodeObject(req.Dados, keys, &v); err != nil {
			return domain.TicketDraft{}, err
		}
		d.System, d.Description, d.Steps, d.Severity = v.Sistema, v.Descricao, v.Passos, v.Severidade
	case domain.TicketTypeDeploy:
		var v deployData
		if err := decodeObject(req.Dados, keys, &v); err != nil {
			return domain.TicketDraft{}, err
		}
		d.Project, d.Repository, d.Environment, d.Notes = v.Projeto, v.Repositorio, v.Ambiente, v.Observacoes
	case domain.TicketTypeOther:
		var v otherData
		if err := decodeObject(req.Dados, keys, &v); err != nil {
			return domain.TicketDraft{}, err
		}
		d.Title, d.Description = v.Titulo, v.Descricao
	default:
		if err := checkKeys(req.Dados, nil); err != nil {
			return domain.TicketDraft{}, err
		}
	}
	return d, nil
}

func decodeObject(raw json.RawMessage, allowed []string, dst any) error {
	if err := checkKeys(raw, allowed); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return invalidBody(reasonMalformed)
	}
	return nil
}

func checkKeys(raw json.RawMessage, allowed []string) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return invalidBody(reasonMalformed)
	}
	seen := make(map[string]bool)
	for dec.More() {
		tok, err := dec.Token()
		key, ok := tok.(string)
		if err != nil || !ok {
			return invalidBody(reasonMalformed)
		}
		folded := strings.ToLower(key)
		if seen[folded] {
			return invalidBody(reasonDuplicateKey)
		}
		seen[folded] = true
		if allowed != nil && !slices.Contains(allowed, key) {
			return invalidBody(reasonMalformed)
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return invalidBody(reasonMalformed)
		}
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return invalidBody(reasonMalformed)
	}
	return nil
}

var ticketFields = map[domain.TicketType][]string{
	domain.TicketTypeAccess: {"sistema", "nivel_acesso", "justificativa"},
	domain.TicketTypeBug:    {"sistema", "descricao", "passos", "severidade"},
	domain.TicketTypeDeploy: {"projeto", "repositorio", "ambiente", "observacoes"},
	domain.TicketTypeOther:  {"titulo", "descricao"},
}

func TicketDataKeys() map[domain.TicketType][]string {
	out := make(map[domain.TicketType][]string, len(ticketFields))
	for typ, keys := range ticketFields {
		out[typ] = slices.Clone(keys)
	}
	return out
}

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
