package security

// Testes do comportamento de negação por padrão (fail closed).

import (
	"context"
	"testing"

	"backend_si/internal/domain"
)

func TestPolicyConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		policy     Policy
		wantZero   bool
		wantPublic bool
		wantRoles  []domain.Role
	}{
		{"valor zero", Policy{}, true, false, nil},
		{"Public", Public(), false, true, nil},
		{"RequireRole admin", RequireRole(domain.RoleAdmin), false, false, []domain.Role{domain.RoleAdmin}},
		{"RequireRole remove duplicados", RequireRole(domain.RoleComum, domain.RoleComum), false, false, []domain.Role{domain.RoleComum}},
		{"RequireRole sem papéis equivale a zero", RequireRole(), true, false, nil},
		{"RequireRole com papel inválido equivale a zero", RequireRole("root"), true, false, nil},
		{"RequireRole descarta inválidos", RequireRole("root", domain.RoleAdmin), false, false, []domain.Role{domain.RoleAdmin}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.policy.IsZero(); got != tt.wantZero {
				t.Errorf("IsZero() = %v, want %v", got, tt.wantZero)
			}
			if got := tt.policy.IsPublic(); got != tt.wantPublic {
				t.Errorf("IsPublic() = %v, want %v", got, tt.wantPublic)
			}
			got := tt.policy.Roles()
			if len(got) != len(tt.wantRoles) {
				t.Fatalf("Roles() = %v, want %v", got, tt.wantRoles)
			}
			for i := range got {
				if got[i] != tt.wantRoles[i] {
					t.Errorf("Roles() = %v, want %v", got, tt.wantRoles)
				}
			}
		})
	}
}

func TestPolicyRolesReturnsCopy(t *testing.T) {
	t.Parallel()

	p := RequireRole(domain.RoleComum)
	p.Roles()[0] = domain.RoleAdmin
	if p.Roles()[0] != domain.RoleComum {
		t.Fatal("Roles() deve retornar uma cópia; a política foi alterada externamente")
	}
}

func TestPrincipalFromRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{"contexto vazio", context.Background()},
		{"usuário zero", context.WithValue(context.Background(), principalKey{}, Principal{Role: domain.RoleAdmin})},
		{"papel inválido", context.WithValue(context.Background(), principalKey{}, Principal{UserID: 1, Role: "root"})},
		{"tipo errado", context.WithValue(context.Background(), principalKey{}, "admin")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, ok := PrincipalFrom(tt.ctx); ok {
				t.Fatal("PrincipalFrom aceitou um principal inválido")
			}
		})
	}
}
