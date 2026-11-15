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

	// Running the seed again fails with a conflict and does not duplicate.
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
