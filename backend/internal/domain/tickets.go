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
// always TicketStatusOpen (BR-1); the other statuses belong to later specs.
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
