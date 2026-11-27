package security

import (
	"fmt"
	"net/http"
)

// Ticket messages (spec 003 section 4, BR-6): the client sends only the
// text; the ticket, the author and the date come from the server (S-8).

// MessageMaxLength is the longest message, in Unicode code points after
// trimming.
const MessageMaxLength = 2000

// messageRule is the multi-line rule of spec 001 section 4 with 1 to
// MessageMaxLength code points: trimmed, valid UTF-8, and line feed,
// carriage return and tab as the only control characters.
var messageRule = multiLine(1, MessageMaxLength)

// reasonNoText is the refusal of a body without texto, or with texto null.
const reasonNoText = "missing texto"

type messageRequest struct {
	Texto *string `json:"texto"`
}

// DecodeMessage reads the body of a reply, {"texto": "…"}, with the rules
// of DecodeJSON: exact keys (any other key, such as an author or a date,
// is refused: S-8), no duplicate, valid UTF-8, at most MaxBodyBytes. A
// missing or null texto is refused too. Every refusal wraps
// domain.ErrInvalidInput (400) with a fixed reason. The text is returned
// raw; ValidateMessage checks it.
func (s *Security) DecodeMessage(w http.ResponseWriter, r *http.Request) (string, error) {
	var req messageRequest
	if err := s.DecodeJSON(w, r, &req); err != nil {
		return "", err
	}
	if req.Texto == nil {
		return "", invalidBody(reasonNoText)
	}
	return *req.Texto, nil
}

// ValidateMessage applies the message rule and returns the text to store,
// trimmed. The error wraps domain.ErrInvalidInput (422 in the handler) with
// a fixed reason and never the text.
func (s *Security) ValidateMessage(raw string) (string, error) {
	text, err := messageRule(raw)
	if err != nil {
		return "", fmt.Errorf("security: validate message: %w", err)
	}
	return text, nil
}
