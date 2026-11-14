package security

import (
	"fmt"

	"backend_si/internal/domain"
)

// Argon2Params são os parâmetros do Argon2id.
type Argon2Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32 // bytes
	KeyLength   uint32 // bytes
}

// DefaultArgon2Params retorna a segunda configuração recomendada pela
// RFC 9106 (64 MiB, t=3, p=4), com sal de 16 bytes e hash de 32 bytes.
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// HashPassword gera o hash Argon2id da senha com sal aleatório
// (crypto/rand) e o codifica no formato PHC:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<sal-base64>$<hash-base64>
func (s *Security) HashPassword(password string) (string, error) {
	// TODO(etapa 2): implementar com golang.org/x/crypto/argon2.IDKey.
	return "", fmt.Errorf("security: hash de senha: %w", domain.ErrNotImplemented)
}

// VerifyPassword compara a senha com um hash PHC em tempo constante.
// Retorna nil somente quando a senha confere; qualquer outro caso
// (incluindo hash malformado) retorna erro que encapsula
// domain.ErrInvalidCredentials.
func (s *Security) VerifyPassword(encodedHash, password string) error {
	// TODO(etapa 2): decodificar o PHC, recalcular com os mesmos parâmetros
	// e comparar com subtle.ConstantTimeCompare.
	//
	// Stub fail-closed: nenhuma senha é aceita.
	return fmt.Errorf("security: verificar senha: %w: %w", domain.ErrInvalidCredentials, domain.ErrNotImplemented)
}
