package security

// Testes do comportamento de negação por padrão (fail closed) da etapa 1.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestAuthorizeDeniesByDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		policy     Policy
		wantCalled bool
	}{
		{"política zero nega", Policy{}, false},
		{"RequireRole() vazio nega", RequireRole(), false},
		{"pública permite", Public(), true},
		{"admin nega (stub)", RequireRole(domain.RoleAdmin), false},
		{"comum nega (stub)", RequireRole(domain.RoleComum), false},
		{"qualquer autenticado nega (stub)", RequireRole(domain.RoleAdmin, domain.RoleComum), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestSecurity(t, nil)
			for _, method := range []string{http.MethodGet, http.MethodPost} {
				var called bool
				rr := httptest.NewRecorder()
				s.Authorize(tt.policy)(okHandler(&called)).ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), method, "/x", http.NoBody))

				if called != tt.wantCalled {
					t.Errorf("%s: handler executado = %v, want %v", method, called, tt.wantCalled)
				}
				if !tt.wantCalled && rr.Code < 300 {
					t.Errorf("%s: negação respondeu %d (sucesso)", method, rr.Code)
				}
			}
		})
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

func TestStubsFailClosed(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)

	tests := []struct {
		name    string
		call    func() error
		wantErr []error
	}{
		{"ParseRole admin", func() error { _, err := s.ParseRole("admin"); return err }, []error{domain.ErrInvalidInput}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.call()
			if err == nil {
				t.Fatal("stub retornou sucesso; deveria falhar fechado")
			}
			for _, want := range tt.wantErr {
				if !errors.Is(err, want) {
					t.Errorf("erro %v não encapsula %v", err, want)
				}
			}
		})
	}
}
