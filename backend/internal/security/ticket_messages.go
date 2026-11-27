package security

import (
	"fmt"
	"net/http"
)

// MessageMaxLength is in Unicode code points after trimming.
const MessageMaxLength = 2000

var messageRule = multiLine(1, MessageMaxLength)

const reasonNoText = "missing texto"

type messageRequest struct {
	Texto *string `json:"texto"`
}

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

func (s *Security) ValidateMessage(raw string) (string, error) {
	text, err := messageRule(raw)
	if err != nil {
		return "", fmt.Errorf("security: validate message: %w", err)
	}
	return text, nil
}
