package domain

import "time"

// Ticket types, statuses and fixed-choice values. The codes are what the
// API and the database store; the labels the user sees live in the frontend.

type TicketID int64

type TicketType string

const (
	TicketTypeAccess TicketType = "acesso"
	TicketTypeBug    TicketType = "bug"
	TicketTypeDeploy TicketType = "deploy"
	TicketTypeOther  TicketType = "outro"
)

func (t TicketType) Valid() bool {
	switch t {
	case TicketTypeAccess, TicketTypeBug, TicketTypeDeploy, TicketTypeOther:
		return true
	}
	return false
}

// TicketStatus lists every status the database accepts. A new ticket is
// always TicketStatusOpen (spec 001 BR-1); only admins move it to the other
// statuses, along the transitions of spec 003 BR-3, which the security
// package and the database enforce. TicketStatusClosed is final.
type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "aberto"
	TicketStatusInProgress TicketStatus = "em_andamento"
	TicketStatusResolved   TicketStatus = "resolvido"
	TicketStatusClosed     TicketStatus = "fechado"
)

func (s TicketStatus) Valid() bool {
	switch s {
	case TicketStatusOpen, TicketStatusInProgress, TicketStatusResolved, TicketStatusClosed:
		return true
	}
	return false
}

type AccessLevel string

const (
	AccessLevelRead  AccessLevel = "leitura"
	AccessLevelWrite AccessLevel = "escrita"
	AccessLevelAdmin AccessLevel = "administrador"
)

func (a AccessLevel) Valid() bool {
	switch a {
	case AccessLevelRead, AccessLevelWrite, AccessLevelAdmin:
		return true
	}
	return false
}

type Severity string

const (
	SeverityLow      Severity = "baixa"
	SeverityMedium   Severity = "media"
	SeverityHigh     Severity = "alta"
	SeverityCritical Severity = "critica"
)

func (s Severity) Valid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	}
	return false
}

type Environment string

const (
	EnvironmentDevelopment Environment = "dev"
	EnvironmentStaging     Environment = "homolog"
	EnvironmentProduction  Environment = "prod"
)

func (e Environment) Valid() bool {
	switch e {
	case EnvironmentDevelopment, EnvironmentStaging, EnvironmentProduction:
		return true
	}
	return false
}

// TicketDraft is a ticket request as decoded, before any validation: every
// value is the raw string the client sent, and a field the type does not
// have stays empty. Notes is nil when the client did not send it.
type TicketDraft struct {
	Type          TicketType
	Title         string
	System        string
	AccessLevel   string
	Justification string
	Description   string
	Steps         string
	Severity      string
	Project       string
	Repository    string
	Environment   string
	Notes         *string
}

// Ticket is a validated ticket. Text fields are trimmed; a field the type
// does not have, and empty notes, are "" and stored as NULL. AuthorID,
// Status and the dates never come from the client (BR-1, BR-2).
type Ticket struct {
	ID            TicketID
	AuthorID      UserID
	Type          TicketType
	Status        TicketStatus
	Title         string
	System        string
	AccessLevel   AccessLevel
	Justification string
	Description   string
	Steps         string
	Severity      Severity
	Project       string
	Repository    string
	Environment   Environment
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TicketSummary is a row of "Meus tickets" and of the admin queue (spec 003
// section 4): never the fields of the type nor the author's id.
// AuthorLogin is filled for the queue only.
type TicketSummary struct {
	ID          TicketID
	Type        TicketType
	Status      TicketStatus
	Title       string
	AuthorLogin string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TicketListQuery is a validated list request. An empty Status, Type or
// AuthorLogin means "all"; AuthorLogin is only set for the queue, never
// for the own list (S-3). Page starts at 1.
type TicketListQuery struct {
	Status      TicketStatus
	Type        TicketType
	AuthorLogin string
	Page        int
}

// TicketPage is one page of a list, ordered by the last update (BR-8).
// Total counts every ticket matching the query; a page past the end has
// no items and keeps the real Total and Pages.
type TicketPage struct {
	Items []TicketSummary
	Total int
	Page  int
	Pages int
}

// StatusCounts is the number of tickets in each status over the whole
// queue, whatever the filters (AC-4.5).
type StatusCounts struct {
	Open       int
	InProgress int
	Resolved   int
	Closed     int
}

type TicketEntryID int64

// EntryKind tells the two kinds of a ticket's history apart (BR-4): a
// message, or a status change made by an admin.
type EntryKind string

const (
	EntryKindMessage      EntryKind = "message"
	EntryKindStatusChange EntryKind = "status_change"
)

func (k EntryKind) Valid() bool {
	return k == EntryKindMessage || k == EntryKindStatusChange
}

// TicketEntry is one entry of a ticket's thread, never edited or deleted
// (BR-4). A message has Body and no statuses; a status change has From and
// To and no Body. The actor, the ticket and the date are set by the server
// (S-8). ActorLogin and ActorRole are read with the entry; who may see the
// login is decided by the web layer (S-13).
type TicketEntry struct {
	ID         TicketEntryID
	TicketID   TicketID
	ActorID    UserID
	ActorLogin string
	ActorRole  Role
	Kind       EntryKind
	Body       string
	From       TicketStatus
	To         TicketStatus
	CreatedAt  time.Time
}

// ThreadPage is a page of a thread, oldest first. HasMore is true when
// there are entries older than the first one.
type ThreadPage struct {
	Entries []TicketEntry
	HasMore bool
}
