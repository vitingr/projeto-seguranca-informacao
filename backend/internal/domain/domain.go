// Package domain contém as entidades e os erros de domínio da aplicação.
//
// É o núcleo da arquitetura: não importa nenhum outro pacote interno.
package domain

import "time"

// Role é o papel de um usuário no controle de acesso.
type Role string

// Papéis válidos. Devem permanecer em sincronia com o CHECK da tabela users.
const (
	RoleAdmin Role = "admin"
	RoleComum Role = "comum"
)

// Valid informa se r é um dos papéis conhecidos.
func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleComum
}

// UserID identifica um usuário. O valor zero significa "nenhum usuário".
type UserID int64

// User é uma conta da aplicação.
type User struct {
	ID           UserID
	Login        string
	PasswordHash string // hash Argon2id em formato PHC; nunca a senha em texto claro
	Role         Role
	CreatedAt    time.Time
}

// Session é uma sessão mantida no servidor.
//
// O token da sessão só existe no cookie do navegador; o banco guarda apenas
// o seu hash (TokenHash), de modo que um vazamento da tabela não permite
// sequestrar sessões.
type Session struct {
	TokenHash  []byte
	UserID     UserID // zero quando a sessão é anônima (pré-login)
	Role       Role   // papel atual do usuário (obtido por JOIN); vazio se anônima
	CSRFToken  string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

// Authenticated informa se a sessão pertence a um usuário autenticado.
func (s Session) Authenticated() bool {
	return s.UserID != 0
}
