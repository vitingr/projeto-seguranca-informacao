package web

import (
	"context"
	"errors"

	"backend_si/internal/domain"
)

// The spec 003 methods of service.TicketRepository on the fake of
// tickets_test.go, so that file stays as it is. The ticket tracking routes
// do not exist yet; their tests use fakes of their own.

var errNotInWebFake = errors.New("not implemented by this fake")

func (m *memTicketRepo) Get(context.Context, domain.TicketID) (domain.Ticket, string, error) {
	return domain.Ticket{}, "", errNotInWebFake
}

func (m *memTicketRepo) ListOwn(context.Context, domain.UserID, domain.TicketListQuery) (domain.TicketPage, error) {
	return domain.TicketPage{}, errNotInWebFake
}

func (m *memTicketRepo) ListQueue(context.Context, domain.TicketListQuery) (domain.TicketPage, domain.StatusCounts, error) {
	return domain.TicketPage{}, domain.StatusCounts{}, errNotInWebFake
}

func (m *memTicketRepo) QueueAuthors(context.Context, int) ([]string, error) {
	return nil, errNotInWebFake
}
