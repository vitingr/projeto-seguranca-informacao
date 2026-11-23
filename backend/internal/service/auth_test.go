package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"backend_si/internal/domain"
)

const (
	adminLogin    = "admin"
	adminPassword = "a long enough admin password"
)

// newAuthWithAdmin returns an Auth whose repository holds one admin, created
// through CreateFirstAdmin with the real hasher, and the spy around it.
func newAuthWithAdmin(t *testing.T) (*Auth, *memUsers, *spyHasher) {
	t.Helper()
	sec := newSecurity(t)
	repo := newMemUsers()
	if _, err := NewUsers(repo, sec, sec).CreateFirstAdmin(t.Context(), adminLogin, adminPassword); err != nil {
		t.Fatalf("CreateFirstAdmin: %v", err)
	}
	spy := &spyHasher{PasswordHasher: sec}
	auth, err := NewAuth(t.Context(), repo, spy, sec)
	if err != nil {
		t.Fatalf("NewAuth: %v", err)
	}
	return auth, repo, spy
}

func TestNewAuthComputesDummyHash(t *testing.T) {
	t.Parallel()

	sec := newSecurity(t)
	spy := &spyHasher{PasswordHasher: sec}
	auth, err := NewAuth(t.Context(), newMemUsers(), spy, sec)
	if err != nil {
		t.Fatalf("NewAuth: %v", err)
	}
	if spy.hashCalls != 1 {
		t.Errorf("HashPassword called %d times, want 1", spy.hashCalls)
	}
	if !strings.HasPrefix(auth.dummyHash, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Errorf("dummy hash %q does not use the current parameters", auth.dummyHash)
	}

	spy.hashErr = context.DeadlineExceeded
	if _, err := NewAuth(t.Context(), newMemUsers(), spy, sec); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("NewAuth with a failing hasher = %v, want the hasher error", err)
	}
}

func TestAuthenticate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		login    string
		password string
		wantOK   bool
	}{
		{"valid credentials", adminLogin, adminPassword, true},
		{"login is case insensitive", "ADMIN", adminPassword, true},
		{"wrong password", adminLogin, adminPassword + "!", false},
		{"unknown user", "nobody", adminPassword, false},
		{"empty login", "", adminPassword, false},
		{"login outside the creation rule is not validated", "x y'--", adminPassword, false},
		{"empty password", adminLogin, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			auth, _, spy := newAuthWithAdmin(t)
			u, err := auth.Authenticate(t.Context(), tt.login, tt.password)

			// D-13: one Argon2id verification in every case.
			if n := len(spy.verifies()); n != 1 {
				t.Errorf("VerifyPassword called %d times, want 1", n)
			}
			if !tt.wantOK {
				if !errors.Is(err, domain.ErrInvalidCredentials) {
					t.Fatalf("Authenticate() = %v, want ErrInvalidCredentials", err)
				}
				if u != (domain.User{}) {
					t.Errorf("user = %+v, want zero", u)
				}
				return
			}
			if err != nil {
				t.Fatalf("Authenticate() = %v", err)
			}
			if u.ID == 0 || u.Login != adminLogin || u.Role != domain.RoleAdmin {
				t.Errorf("user = %+v", u)
			}
			if u.PasswordHash != "" {
				t.Error("Authenticate returned the password hash")
			}
		})
	}
}

// TestAuthenticateUnknownUserLooksLikeWrongPassword is the D-13 criterion:
// both cases verify exactly once (the unknown user against the dummy hash)
// and return the same error.
func TestAuthenticateUnknownUserLooksLikeWrongPassword(t *testing.T) {
	t.Parallel()

	auth, repo, spy := newAuthWithAdmin(t)
	_, errUnknown := auth.Authenticate(t.Context(), "nobody", adminPassword)
	_, errWrong := auth.Authenticate(t.Context(), adminLogin, "a wrong but long password")

	calls := spy.verifies()
	if len(calls) != 2 {
		t.Fatalf("VerifyPassword called %d times, want 1 per attempt", len(calls))
	}
	if calls[0] != auth.dummyHash {
		t.Error("the unknown user was not verified against the dummy hash")
	}
	admin, _ := repo.GetByLogin(t.Context(), adminLogin)
	if calls[1] != admin.PasswordHash {
		t.Error("the wrong password was not verified against the stored hash")
	}
	if errUnknown == nil || errWrong == nil || errUnknown.Error() != errWrong.Error() {
		t.Errorf("errors differ: %v / %v", errUnknown, errWrong)
	}
}

// TestAuthenticateServerFailures checks that failures other than bad
// credentials are not reported as 401: they must not wrap
// ErrInvalidCredentials.
func TestAuthenticateServerFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		login     string
		repoErr   error
		verifyErr error
		want      error
	}{
		{"repository failure", adminLogin, errDatabase, nil, errDatabase},
		{"hash wait ends, known user", adminLogin, nil, context.DeadlineExceeded, context.DeadlineExceeded},
		{"hash wait ends, unknown user", "nobody", nil, context.DeadlineExceeded, context.DeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			auth, repo, spy := newAuthWithAdmin(t)
			repo.err = tt.repoErr
			spy.verifyErr = tt.verifyErr

			_, err := auth.Authenticate(t.Context(), tt.login, adminPassword)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Authenticate() = %v, want %v", err, tt.want)
			}
			if errors.Is(err, domain.ErrInvalidCredentials) {
				t.Errorf("Authenticate() = %v must not be a credential error", err)
			}
		})
	}
}

// TestAuthenticateNamesTheTargetedUser covers the failed-login target of
// 006 D-4: a wrong password names the existing user, an unknown login names
// nobody, and neither returns a user.
func TestAuthenticateNamesTheTargetedUser(t *testing.T) {
	t.Parallel()

	auth, repo, _ := newAuthWithAdmin(t)
	admin, err := repo.GetByLogin(t.Context(), adminLogin)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		login  string
		wantID domain.UserID
	}{
		{"wrong password", "ADMIN", admin.ID},
		{"unknown login", "nobody", 0},
	}
	for _, tt := range tests {
		u, err := auth.Authenticate(t.Context(), tt.login, "a wrong but long password")
		var credErr *domain.InvalidCredentialsError
		if !errors.As(err, &credErr) || !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("%s: error = %v, want an InvalidCredentialsError", tt.name, err)
		}
		if credErr.UserID != tt.wantID || u != (domain.User{}) {
			t.Errorf("%s: target %d, user %+v; want target %d and no user", tt.name, credErr.UserID, u, tt.wantID)
		}
	}
}
