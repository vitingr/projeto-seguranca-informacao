package security

import (
	"net/http"
	"slices"

	"backend_si/internal/domain"
)

type Policy struct {
	public bool
	roles  []domain.Role
}

func Public() Policy {
	return Policy{public: true}
}

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

func (p Policy) IsZero() bool {
	return !p.public && len(p.roles) == 0
}

func (p Policy) IsPublic() bool {
	return p.public
}

func (p Policy) Roles() []domain.Role {
	return slices.Clone(p.roles)
}

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
