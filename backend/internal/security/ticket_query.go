package security

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"backend_si/internal/domain"
)

// Query parameters of the ticket lists and of the thread (spec 003 section
// 4, plan P-9). Every rule is strict: an unknown, repeated or empty
// parameter, or a value outside its rule, is ErrInvalidInput (400) with a
// fixed reason that never quotes the client's text.

// TicketListScope says which list a query is for.
type TicketListScope int

const (
	// TicketListOwn is "Meus tickets": the author is always the session
	// user, so a usuario parameter is refused (S-3).
	TicketListOwn TicketListScope = iota + 1
	// TicketListQueue is the admin queue, which may filter by author.
	TicketListQueue
)

// MaxTicketListPage is the last page a list query may ask for.
const MaxTicketListPage = 10000

// Names of the list and thread parameters, the same as in the page address
// (AC-1.6).
const (
	paramStatus = "status"
	paramType   = "tipo"
	paramAuthor = "usuario"
	paramPage   = "pagina"
	paramBefore = "antes"
)

const (
	queryMalformed     = "malformed query"
	queryUnknownParam  = "unknown parameter"
	queryRepeatedParam = "repeated parameter"
	queryEmptyValue    = "empty value"
	queryInvalidStatus = "invalid status"
	queryInvalidType   = "invalid type"
	queryInvalidLogin  = "invalid login"
	queryAuthorRefused = "author filter not allowed"
	queryInvalidPage   = "invalid page"
	queryInvalidCursor = "invalid cursor"
)

func invalidQuery(reason string) error {
	return fmt.Errorf("security: ticket query: %s: %w", reason, domain.ErrInvalidInput)
}

// errInvalidScope is a programming error (a scope that is neither list):
// it is not ErrInvalidInput, so the client gets a 500 (fail closed).
var errInvalidScope = errors.New("security: ticket query: invalid scope")

// ReadQuery parses the query string of r strictly. Unlike r.URL.Query(),
// which drops the pairs it cannot parse, a malformed query (a bad escape, a
// semicolon) is an error, so nothing the client sent is silently ignored.
func (s *Security) ReadQuery(r *http.Request) (url.Values, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, invalidQuery(queryMalformed)
	}
	return values, nil
}

// ParseTicketListQuery validates the parameters of a ticket list: status
// and tipo (one code each), usuario (a login, queue only) and pagina (1 to
// MaxTicketListPage, digits only, no leading zero). Absent filters mean
// "all"; an absent page is page 1.
func (s *Security) ParseTicketListQuery(values url.Values, scope TicketListScope) (domain.TicketListQuery, error) {
	allowed := []string{paramStatus, paramType, paramPage}
	switch scope {
	case TicketListOwn:
	case TicketListQueue:
		allowed = append(allowed, paramAuthor)
	default:
		return domain.TicketListQuery{}, errInvalidScope
	}
	if scope == TicketListOwn && len(values[paramAuthor]) > 0 {
		return domain.TicketListQuery{}, invalidQuery(queryAuthorRefused)
	}
	single, err := singleValues(values, allowed)
	if err != nil {
		return domain.TicketListQuery{}, err
	}

	q := domain.TicketListQuery{Page: 1}
	if raw, ok := single[paramStatus]; ok {
		q.Status = domain.TicketStatus(raw)
		if !q.Status.Valid() {
			return domain.TicketListQuery{}, invalidQuery(queryInvalidStatus)
		}
	}
	if raw, ok := single[paramType]; ok {
		q.Type = domain.TicketType(raw)
		if !q.Type.Valid() {
			return domain.TicketListQuery{}, invalidQuery(queryInvalidType)
		}
	}
	if raw, ok := single[paramAuthor]; ok {
		if s.ValidateLogin(raw) != nil {
			return domain.TicketListQuery{}, invalidQuery(queryInvalidLogin)
		}
		q.AuthorLogin = raw
	}
	if raw, ok := single[paramPage]; ok {
		page, ok := parsePositive(raw)
		if !ok || page > MaxTicketListPage {
			return domain.TicketListQuery{}, invalidQuery(queryInvalidPage)
		}
		q.Page = int(page)
	}
	return q, nil
}

// ParseThreadCursor validates the only parameter of a thread request,
// antes: the id of the oldest entry already shown, as a positive int64.
// Absent, it is 0, which means the latest entries (plan P-4).
func (s *Security) ParseThreadCursor(values url.Values) (domain.TicketEntryID, error) {
	single, err := singleValues(values, []string{paramBefore})
	if err != nil {
		return 0, err
	}
	raw, ok := single[paramBefore]
	if !ok {
		return 0, nil
	}
	id, ok := parsePositive(raw)
	if !ok {
		return 0, invalidQuery(queryInvalidCursor)
	}
	return domain.TicketEntryID(id), nil
}

// singleValues checks that every parameter is one of allowed, given once,
// with a non-empty value, and returns the values by name.
func singleValues(values url.Values, allowed []string) (map[string]string, error) {
	single := make(map[string]string, len(values))
	for name, vs := range values {
		if !slices.Contains(allowed, name) {
			return nil, invalidQuery(queryUnknownParam)
		}
		if len(vs) != 1 {
			return nil, invalidQuery(queryRepeatedParam)
		}
		if vs[0] == "" {
			return nil, invalidQuery(queryEmptyValue)
		}
		single[name] = vs[0]
	}
	return single, nil
}

// parsePositive reads a positive int64 written with the ticket id rule:
// ASCII digits only, no sign, no leading zero (ParseTicketID).
func parsePositive(raw string) (int64, bool) {
	id, ok := ParseTicketID(raw)
	return int64(id), ok
}
