package domain

import "time"

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

type TicketSummary struct {
	ID          TicketID
	Type        TicketType
	Status      TicketStatus
	Title       string
	AuthorLogin string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TicketListQuery struct {
	Status      TicketStatus
	Type        TicketType
	AuthorLogin string
	Page        int
}

type TicketPage struct {
	Items []TicketSummary
	Total int
	Page  int
	Pages int
}

type StatusCounts struct {
	Open       int
	InProgress int
	Resolved   int
	Closed     int
}

type TicketEntryID int64

type EntryKind string

const (
	EntryKindMessage      EntryKind = "message"
	EntryKindStatusChange EntryKind = "status_change"
)

func (k EntryKind) Valid() bool {
	return k == EntryKindMessage || k == EntryKindStatusChange
}

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

type ThreadPage struct {
	Entries []TicketEntry
	HasMore bool
}
