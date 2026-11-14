package security

import (
	"fmt"
	"net/http"

	"backend_si/internal/domain"
)

// MaxBodyBytes limita o corpo das requisições JSON aceitas (proteção contra DoS).
const MaxBodyBytes = 64 << 10

// DecodeJSON lê o corpo da requisição em dst com as seguintes regras:
// Content-Type application/json obrigatório, corpo limitado a MaxBodyBytes,
// campos desconhecidos rejeitados e um único objeto JSON por requisição.
// Qualquer violação retorna erro que encapsula domain.ErrInvalidInput.
func (s *Security) DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// TODO(etapa 2): checar Content-Type, http.MaxBytesReader,
	// json.Decoder com DisallowUnknownFields e garantir que não há dados
	// após o primeiro objeto.
	return fmt.Errorf("security: ler JSON: %w: %w", domain.ErrInvalidInput, domain.ErrNotImplemented)
}

// ValidateLogin valida o login por lista de permissão (allowlist) de
// caracteres e tamanho.
func (s *Security) ValidateLogin(login string) error {
	// TODO(etapa 2): definir a regra (ex.: ^[a-z0-9._-]{3,64}$).
	return fmt.Errorf("security: validar login: %w: %w", domain.ErrInvalidInput, domain.ErrNotImplemented)
}

// ValidatePassword aplica a política de senha (tamanho mínimo e máximo).
func (s *Security) ValidatePassword(password string) error {
	// TODO(etapa 2): definir a política (ex.: 12 a 128 caracteres).
	return fmt.Errorf("security: validar senha: %w: %w", domain.ErrInvalidInput, domain.ErrNotImplemented)
}

// ParseRole converte o valor recebido na requisição em um papel válido.
func (s *Security) ParseRole(raw string) (domain.Role, error) {
	// TODO(etapa 2): aceitar somente valores para os quais domain.Role.Valid é verdadeiro.
	return "", fmt.Errorf("security: validar papel: %w: %w", domain.ErrInvalidInput, domain.ErrNotImplemented)
}
