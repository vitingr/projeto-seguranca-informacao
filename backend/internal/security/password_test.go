package security

import (
	"context"
	"errors"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"backend_si/internal/domain"
)

// Reference vectors from an independent Argon2id implementation (Node.js 24).
const (
	refPassword    = "correct horse battery staple"
	refHash        = "$argon2id$v=19$m=65536,t=3,p=4$MDEyMzQ1Njc4OWFiY2RlZg$77UfmnZYT23WpPeUKhovauWm5OxRQv9nTf1dJ+tF5EY"
	refUnicodeHash = "$argon2id$v=19$m=65536,t=3,p=4$MDEyMzQ1Njc4OWFiY2RlZg$no9HIRHOfcXvZ361fis8b0RUx3l1fxfwomsHab6mqbY"
)

var phcPattern = regexp.MustCompile(`^\$argon2id\$v=19\$m=65536,t=3,p=4\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`)

func TestHashPasswordFormat(t *testing.T) {
	t.Parallel()

	s := newTestSecurity(t, nil)
	const password = "a long enough password"
	h1, err := s.HashPassword(t.Context(), password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	h2, err := s.HashPassword(t.Context(), password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	for _, h := range []string{h1, h2} {
		if !phcPattern.MatchString(h) {
			t.Errorf("hash %q is not $argon2id$v=19$m=65536,t=3,p=4$<salt>$<key>", h)
		}
		if strings.Contains(h, password) {
			t.Errorf("hash %q contains the password", h)
		}
	}
	if h1 == h2 {
		t.Error("two hashes of the same password are equal; the salt is not random")
	}
	for _, h := range []string{h1, h2} {
		if err := s.VerifyPassword(t.Context(), h, password); err != nil {
			t.Errorf("VerifyPassword(own hash) = %v", err)
		}
	}
}

func TestVerifyPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hash     string
		password string
		match    bool
	}{
		{"reference vector", refHash, refPassword, true},
		{"wrong password", refHash, refPassword + "!", false},
		{"case matters", refHash, strings.ToUpper(refPassword), false},
		{"empty password", refHash, "", false},
		{"unicode vector, NFKC input", refUnicodeHash, "caf\u00e9 pa\u00dfword 123", true},
		{"unicode vector, equivalent input", refUnicodeHash, "cafe\u0301 pa\u00dfword \uff11\uff12\uff13", true},
		{"unicode vector, different input", refUnicodeHash, "cafe pa\u00dfword 123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestSecurity(t, nil)
			err := s.VerifyPassword(t.Context(), tt.hash, tt.password)
			if tt.match && err != nil {
				t.Errorf("VerifyPassword() = %v, want nil", err)
			}
			if !tt.match && !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Errorf("VerifyPassword() = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestVerifyPasswordRejectsBadHashes(t *testing.T) {
	t.Parallel()

	const (
		salt = "MDEyMzQ1Njc4OWFiY2RlZg"                      // 16 bytes
		key  = "77UfmnZYT23WpPeUKhovauWm5OxRQv9nTf1dJ+tF5EY" // 32 bytes
	)
	phc := func(alg, version, params, salt, key string) string {
		return "$" + alg + "$" + version + "$" + params + "$" + salt + "$" + key
	}
	const ok = "m=65536,t=3,p=4"

	tests := []struct{ name, hash string }{
		{"empty", ""},
		{"garbage", "not a hash"},
		{"bcrypt", "$2b$12$abcdefghijklmnopqrstuuabcdefghijklmnopqrstuvwxyz012"},
		{"argon2i", phc("argon2i", "v=19", ok, salt, key)},
		{"argon2d", phc("argon2d", "v=19", ok, salt, key)},
		{"version 16", phc("argon2id", "v=16", ok, salt, key)},
		{"missing version", "$argon2id$" + ok + "$" + salt + "$" + key},
		{"extra field", refHash + "$x"},
		{"trailing dollar", refHash + "$"},
		{"no leading dollar", strings.TrimPrefix(refHash, "$")},
		{"memory above 256 MiB", phc("argon2id", "v=19", "m=262145,t=3,p=4", salt, key)},
		{"memory 4 GiB", phc("argon2id", "v=19", "m=4294967295,t=3,p=4", salt, key)},
		{"memory overflows uint32", phc("argon2id", "v=19", "m=4294967296,t=3,p=4", salt, key)},
		{"memory below 8*p", phc("argon2id", "v=19", "m=31,t=3,p=4", salt, key)},
		{"zero memory", phc("argon2id", "v=19", "m=0,t=3,p=4", salt, key)},
		{"iterations above 10", phc("argon2id", "v=19", "m=65536,t=11,p=4", salt, key)},
		{"zero iterations", phc("argon2id", "v=19", "m=65536,t=0,p=4", salt, key)},
		{"parallelism above 16", phc("argon2id", "v=19", "m=65536,t=3,p=17", salt, key)},
		{"parallelism 255", phc("argon2id", "v=19", "m=65536,t=3,p=255", salt, key)},
		{"zero parallelism", phc("argon2id", "v=19", "m=65536,t=3,p=0", salt, key)},
		{"negative number", phc("argon2id", "v=19", "m=-1,t=3,p=4", salt, key)},
		{"plus sign", phc("argon2id", "v=19", "m=+65536,t=3,p=4", salt, key)},
		{"leading zero", phc("argon2id", "v=19", "m=065536,t=3,p=4", salt, key)},
		{"reordered params", phc("argon2id", "v=19", "t=3,m=65536,p=4", salt, key)},
		{"missing param", phc("argon2id", "v=19", "m=65536,t=3", salt, key)},
		{"extra param", phc("argon2id", "v=19", ok+",x=1", salt, key)},
		{"spaces", phc("argon2id", "v=19", "m=65536, t=3, p=4", salt, key)},
		{"salt of 15 bytes", phc("argon2id", "v=19", ok, "MDEyMzQ1Njc4OWFiY2Rl", key)},
		{"salt of 65 bytes", phc("argon2id", "v=19", ok, strings.Repeat("QUFB", 21)+"QUE", key)},
		{"key of 15 bytes", phc("argon2id", "v=19", ok, salt, "MDEyMzQ1Njc4OWFiY2Rl")},
		{"key of 65 bytes", phc("argon2id", "v=19", ok, salt, strings.Repeat("QUFB", 21)+"QUE")},
		{"empty salt", phc("argon2id", "v=19", ok, "", key)},
		{"padded base64", phc("argon2id", "v=19", ok, salt+"==", key)},
		{"url base64", phc("argon2id", "v=19", ok, salt, strings.ReplaceAll(key, "+", "-"))},
		{"non-canonical base64", phc("argon2id", "v=19", ok, salt, key[:len(key)-1]+"F")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestSecurity(t, nil)
			err := s.VerifyPassword(t.Context(), tt.hash, refPassword)
			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("VerifyPassword() = %v, want ErrInvalidCredentials", err)
			}
			if !errors.Is(err, errBadHash) {
				t.Errorf("VerifyPassword() = %v, want the malformed-hash error (Argon2id must not run)", err)
			}
		})
	}
}

func TestNewSizesHashSemaphore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opt  int
		want int
	}{
		{"zero means GOMAXPROCS", 0, runtime.GOMAXPROCS(0)},
		{"explicit", 3, 3},
	}
	for _, tt := range tests {
		s := newTestSecurityWith(t, func(o *Options) { o.MaxConcurrentHashes = tt.opt })
		if got := cap(s.hashSlots); got != tt.want {
			t.Errorf("%s: semaphore size = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestHashWaitStopsWithContext(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name string
		call func(ctx context.Context, s *Security) error
	}{
		{"HashPassword", func(ctx context.Context, s *Security) error {
			_, err := s.HashPassword(ctx, refPassword)
			return err
		}},
		{"VerifyPassword", func(ctx context.Context, s *Security) error {
			return s.VerifyPassword(ctx, refHash, refPassword)
		}},
	}
	for _, c := range calls {
		t.Run(c.name+" deadline while waiting", func(t *testing.T) {
			t.Parallel()
			s := newTestSecurityWith(t, func(o *Options) { o.MaxConcurrentHashes = 1 })
			s.hashSlots <- struct{}{} // the only slot is busy
			defer s.releaseHashSlot()

			const wait = 100 * time.Millisecond
			ctx, cancel := context.WithTimeout(t.Context(), wait)
			defer cancel()
			start := time.Now()
			err := c.call(ctx, s)
			elapsed := time.Since(start)

			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want context.DeadlineExceeded", err)
			}
			if errors.Is(err, domain.ErrInvalidCredentials) {
				t.Errorf("error = %v must not be a credential error (it is a 500)", err)
			}
			if elapsed < wait || elapsed > wait+time.Second {
				t.Errorf("returned after %v, want about %v", elapsed, wait)
			}
		})
		t.Run(c.name+" canceled before the call", func(t *testing.T) {
			t.Parallel()
			s := newTestSecurity(t, nil) // free slots
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if err := c.call(ctx, s); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
			if n := len(s.hashSlots); n != 0 {
				t.Errorf("%d slots still taken", n)
			}
		})
		t.Run(c.name+" releases the slot", func(t *testing.T) {
			t.Parallel()
			s := newTestSecurityWith(t, func(o *Options) { o.MaxConcurrentHashes = 1 })
			for range 2 { // the second call blocks forever if the slot leaks
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				err := c.call(ctx, s)
				cancel()
				if err != nil {
					t.Fatalf("error = %v", err)
				}
			}
		})
	}
}

func TestHashSemaphoreBoundsConcurrency(t *testing.T) {
	t.Parallel()

	s := newTestSecurityWith(t, func(o *Options) { o.MaxConcurrentHashes = 1 })
	s.hashSlots <- struct{}{}

	done := make(chan error, 3)
	for range 3 {
		go func() { done <- s.VerifyPassword(t.Context(), refHash, refPassword) }()
	}
	select {
	case err := <-done:
		t.Fatalf("a hash ran while the only slot was busy (err = %v)", err)
	case <-time.After(200 * time.Millisecond):
	}

	s.releaseHashSlot()
	for range 3 {
		if err := <-done; err != nil {
			t.Errorf("VerifyPassword() = %v", err)
		}
	}
}
