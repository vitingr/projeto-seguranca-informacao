package security

import (
	"net/http"
	"slices"

	"backend_si/internal/domain"
)

// Policy descreve quem pode acessar uma rota.
//
// O valor zero (Policy{}) NEGA TUDO: não é pública e não aceita nenhum
// papel. Políticas só são criadas por Public ou RequireRole.
type Policy struct {
	public bool
	roles  []domain.Role
}

// Public permite acesso a qualquer visitante, autenticado ou não.
func Public() Policy {
	return Policy{public: true}
}

// RequireRole exige um usuário autenticado com um dos papéis informados.
// Papéis inválidos são descartados; sem nenhum papel válido o resultado é
// equivalente ao valor zero e nega tudo.
func RequireRole(roles ...domain.Role) Policy {
	valid := make([]domain.Role, 0, len(roles))
	for _, r := range roles {
		if r.Valid() && !slices.Contains(valid, r) {
			valid = append(valid, r)
		}
	}
	if len(valid) == 0 {
		return Policy{}
	}
	return Policy{roles: valid}
}

// IsZero informa se p é o valor zero (nega tudo).
func (p Policy) IsZero() bool {
	return !p.public && len(p.roles) == 0
}

// IsPublic informa se a política libera visitantes anônimos.
func (p Policy) IsPublic() bool {
	return p.public
}

// Roles retorna uma cópia dos papéis aceitos.
func (p Policy) Roles() []domain.Role {
	return slices.Clone(p.roles)
}

// Authorize retorna o middleware que aplica a política p a uma rota.
//
// Regras (negação por padrão):
//   - política zero: nega sempre;
//   - pública: permite;
//   - por papel: exige Principal na requisição com papel contido em p.
//     Anônimo recebe 401 (o frontend leva ao login); papel errado recebe 403.
func (s *Security) Authorize(p Policy) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if p.IsZero() {
				s.deny(w, r, http.StatusForbidden, "authz: rota sem política")
				return
			}
			if p.public {
				next.ServeHTTP(w, r)
				return
			}
			principal, ok := PrincipalFrom(r.Context())
			if !ok {
				s.deny(w, r, http.StatusUnauthorized, "authz: anonymous")
				return
			}
			if !slices.Contains(p.roles, principal.Role) {
				s.deny(w, r, http.StatusForbidden, "authz: role not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
