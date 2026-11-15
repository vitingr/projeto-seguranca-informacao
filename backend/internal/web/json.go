package web

import (
	"encoding/json"
	"net/http"
)

// errorResponse é o corpo de toda resposta de erro. Nunca carrega detalhes
// internos: apenas um código estável, uma mensagem genérica e o request ID
// para correlação com os logs.
type errorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteError responde o erro genérico do status informado. Tem a assinatura
// de security.ErrorRenderer e é injetada no pacote security pelo
// composition root.
func WriteError(w http.ResponseWriter, r *http.Request, status int) {
	writeErrorMessage(w, r, status, defaultMessage(status))
}

// writeErrorMessage responde um erro com mensagem específica e segura para o
// usuário (nunca err.Error()).
func writeErrorMessage(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error:     errorCode(status),
		Message:   message,
		RequestID: RequestIDFrom(r.Context()),
	})
}

// writeJSON serializa v antes de escrever qualquer byte, para que uma falha
// nunca produza uma resposta pela metade.
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
	default:
		return "Ocorreu um erro inesperado. Tente novamente mais tarde."
	}
}
