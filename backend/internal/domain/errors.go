package domain

import "errors"

// Erros de domínio. As camadas superiores os comparam com errors.Is e os
// traduzem em respostas genéricas; a mensagem nunca é exibida ao usuário.
var (
	// ErrNotFound indica que o registro procurado não existe.
	ErrNotFound = errors.New("not found")
	// ErrConflict indica violação de unicidade (ex.: login já cadastrado).
	ErrConflict = errors.New("conflict")
	// ErrInvalidInput indica entrada rejeitada pela validação.
	ErrInvalidInput = errors.New("invalid input")
	// ErrInvalidCredentials indica falha de autenticação, sem distinguir
	// usuário inexistente de senha incorreta.
	ErrInvalidCredentials = errors.New("invalid credentials")
)
