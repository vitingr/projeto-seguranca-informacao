package security

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"backend_si/internal/domain"
)

type TicketListScope int

const (
	TicketListOwn TicketListScope = iota + 1
	TicketListQueue
)

const MaxTicketListPage = 10000

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

var errInvalidScope = errors.New("security: ticket query: invalid scope")

func (s *Security) ReadQuery(r *http.Request) (url.Values, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, invalidQuery(queryMalformed)
	}
	return values, nil
}

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

func parsePositive(raw string) (int64, bool) {
	id, ok := ParseTicketID(raw)
	return int64(id), ok
}
