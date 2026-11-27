package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

func TestCreateFirstAdmin(t *testing.T) {
	t.Parallel()

	sec := newSecurity(t)
	repo := newMemUsers()
	users := NewUsers(repo, sec, sec)

	got, err := users.CreateFirstAdmin(t.Context(), "Admin.Root", adminPassword)
	if err != nil {
		t.Fatalf("CreateFirstAdmin() = %v", err)
	}
	if got.ID == 0 || got.Login != "admin.root" || got.Role != domain.RoleAdmin || got.PasswordHash != "" {
		t.Errorf("returned user = %+v", got)
	}

	stored, err := repo.GetByLogin(t.Context(), "admin.root")
	if err != nil {
		t.Fatalf("stored user: %v", err)
	}
	if !strings.HasPrefix(stored.PasswordHash, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Errorf("stored hash %q is not Argon2id PHC", stored.PasswordHash)
	}
	if strings.Contains(stored.PasswordHash, adminPassword) {
		t.Error("the stored hash contains the password")
	}
	if err := sec.VerifyPassword(t.Context(), stored.PasswordHash, adminPassword); err != nil {
		t.Errorf("stored hash does not verify: %v", err)
	}

	for _, login := range []string{"admin.root", "ADMIN.ROOT"} {
		if _, err := users.CreateFirstAdmin(t.Context(), login, adminPassword); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("second CreateFirstAdmin(%q) = %v, want ErrConflict", login, err)
		}
	}
	if n := repo.count(); n != 1 {
		t.Errorf("repository holds %d users, want 1", n)
	}
}

func TestCreateFirstAdminRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, login, password string
	}{
		{"login too short", "ab", adminPassword},
		{"login with spaces", "the admin", adminPassword},
		{"password too short", adminLogin, "short pw"},
		{"empty password", adminLogin, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sec := newSecurity(t)
			repo := newMemUsers()
			spy := &spyHasher{PasswordHasher: sec}

			_, err := NewUsers(repo, spy, sec).CreateFirstAdmin(t.Context(), tt.login, tt.password)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("CreateFirstAdmin() = %v, want ErrInvalidInput", err)
			}
			if spy.hashCalls != 0 || repo.count() != 0 {
				t.Errorf("hashed %d times and stored %d users; validation must come first", spy.hashCalls, repo.count())
			}
			if strings.Contains(err.Error(), tt.password) && tt.password != "" {
				t.Errorf("error %q contains the password", err)
			}
		})
	}
}

func TestCreateFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    domain.Role
		repoErr error
		hashErr error
		want    error
	}{
		{"invalid role", "root", nil, nil, domain.ErrInvalidInput},
		{"hash wait ends", domain.RoleAdmin, nil, context.DeadlineExceeded, context.DeadlineExceeded},
		{"repository failure", domain.RoleComum, errDatabase, nil, errDatabase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sec := newSecurity(t)
			repo := newMemUsers()
			repo.err = tt.repoErr
			spy := &spyHasher{PasswordHasher: sec, hashErr: tt.hashErr}

			_, err := NewUsers(repo, spy, sec).create(t.Context(), adminLogin, adminPassword, tt.role)
			if !errors.Is(err, tt.want) {
				t.Fatalf("create() = %v, want %v", err, tt.want)
			}
		})
	}
}

type spyRepo struct {
	*memUsers
	creates []domain.User
}

func (s *spyRepo) Create(ctx context.Context, u domain.User) (domain.UserID, error) {
	s.creates = append(s.creates, u)
	return s.memUsers.Create(ctx, u)
}

func TestRegister(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, login string
		role        string
		wantLogin   string
		wantRole    domain.Role
	}{
		{"comum", "Maria.Silva", "comum", "maria.silva", domain.RoleComum},
		{"admin", "joao", "admin", "joao", domain.RoleAdmin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sec := newSecurity(t)
			repo := &spyRepo{memUsers: newMemUsers()}
			users := NewUsers(repo, sec, sec)

			got, err := users.Register(t.Context(), RegisterInput{Login: tt.login, Password: adminPassword, Role: tt.role})
			if err != nil {
				t.Fatalf("Register() = %v", err)
			}
			if got.ID == 0 || got.Login != tt.wantLogin || got.Role != tt.wantRole || got.PasswordHash != "" {
				t.Errorf("returned user = %+v", got)
			}
			if len(repo.creates) != 1 {
				t.Fatalf("repository Create called %d times, want 1", len(repo.creates))
			}
			stored := repo.creates[0]
			if stored.Login != tt.wantLogin || stored.Role != tt.wantRole {
				t.Errorf("stored user = %+v", stored)
			}
			if !strings.HasPrefix(stored.PasswordHash, "$argon2id$v=19$") || strings.Contains(stored.PasswordHash, adminPassword) {
				t.Errorf("stored hash %q is not the hasher's output", stored.PasswordHash)
			}
			if err := sec.VerifyPassword(t.Context(), stored.PasswordHash, adminPassword); err != nil {
				t.Errorf("stored hash does not verify: %v", err)
			}
		})
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   RegisterInput
	}{
		{"role root", RegisterInput{Login: adminLogin, Password: adminPassword, Role: "root"}},
		{"role in upper case", RegisterInput{Login: adminLogin, Password: adminPassword, Role: "ADMIN"}},
		{"missing role", RegisterInput{Login: adminLogin, Password: adminPassword}},
		{"login too short", RegisterInput{Login: "ab", Password: adminPassword, Role: "comum"}},
		{"login with spaces", RegisterInput{Login: "the user", Password: adminPassword, Role: "comum"}},
		{"password too short", RegisterInput{Login: adminLogin, Password: "short pw", Role: "comum"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sec := newSecurity(t)
			repo := newMemUsers()
			spy := &spyHasher{PasswordHasher: sec}

			_, err := NewUsers(repo, spy, sec).Register(t.Context(), tt.in)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("Register() = %v, want ErrInvalidInput", err)
			}
			if spy.hashCalls != 0 || repo.count() != 0 {
				t.Errorf("hashed %d times and stored %d users; validation must come first", spy.hashCalls, repo.count())
			}
			if strings.Contains(err.Error(), tt.in.Password) {
				t.Errorf("error %q contains the password", err)
			}
		})
	}
}

func TestRegisterConflict(t *testing.T) {
	t.Parallel()

	sec := newSecurity(t)
	repo := newMemUsers()
	users := NewUsers(repo, sec, sec)
	in := RegisterInput{Login: "maria.silva", Password: adminPassword, Role: "comum"}
	if _, err := users.Register(t.Context(), in); err != nil {
		t.Fatalf("first Register() = %v", err)
	}

	for _, login := range []string{"maria.silva", "Maria.Silva"} {
		in.Login = login
		if _, err := users.Register(t.Context(), in); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("Register(%q) = %v, want ErrConflict", login, err)
		}
	}
	if n := repo.count(); n != 1 {
		t.Errorf("repository holds %d users, want 1", n)
	}
}

func TestRegisterFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repoErr error
		hashErr error
		want    error
	}{
		{"hash wait ends", nil, context.DeadlineExceeded, context.DeadlineExceeded},
		{"repository failure", errDatabase, nil, errDatabase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sec := newSecurity(t)
			repo := newMemUsers()
			repo.err = tt.repoErr
			spy := &spyHasher{PasswordHasher: sec, hashErr: tt.hashErr}

			_, err := NewUsers(repo, spy, sec).Register(t.Context(),
				RegisterInput{Login: adminLogin, Password: adminPassword, Role: "comum"})
			if !errors.Is(err, tt.want) {
				t.Fatalf("Register() = %v, want %v", err, tt.want)
			}
			if errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrConflict) {
				t.Errorf("Register() = %v, must not look like a validation error or a conflict", err)
			}
		})
	}
}
