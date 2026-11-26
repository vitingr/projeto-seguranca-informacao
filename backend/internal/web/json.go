package web

import (
	"encoding/json"
	"net/http"
)

// msgTicketClosed is the 409 of a reply on a "fechado" ticket (spec 003
// BR-5, AC-3.7).
const msgTicketClosed = "Este ticket foi fechado e não aceita novas mensagens."

type errorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func WriteError(w http.ResponseWriter, r *http.Request, status int) {
	writeErrorMessage(w, r, status, defaultMessage(status))
}

func writeErrorMessage(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error:     errorCode(status),
		Message:   message,
		RequestID: RequestIDFrom(r.Context()),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":"internal","message":"Erro interno."}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func errorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case http.StatusTooManyRequests:
		return "too_many_requests"
	default:
		return "internal"
	}
}

func defaultMessage(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "A requisição não pôde ser processada."
	case http.StatusUnauthorized:
		return "Autenticação necessária."
	case http.StatusForbidden:
		return "Você não tem permissão para acessar este recurso."
	case http.StatusNotFound:
		return "Recurso não encontrado."
	case http.StatusConflict:
		return "O recurso já existe."
	case http.StatusUnprocessableEntity:
		return "Dados inválidos. Revise os campos e tente novamente."
	case http.StatusTooManyRequests:
		return "Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente."
	default:
		return "Ocorreu um erro inesperado. Tente novamente mais tarde."
	}
}
