package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"

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

// Limits for the parameters of a stored hash (D-15). A PHC string outside
// them is rejected before any work, so a tampered hash cannot make
// VerifyPassword allocate or compute without bound.
const (
	maxArgon2Memory      = 256 * 1024 // KiB (256 MiB)
	maxArgon2Iterations  = 10
	maxArgon2Parallelism = 16
	minArgon2Bytes       = 16 // salt and key length
	maxArgon2Bytes       = 64
)

// validate reports whether p is within the D-15 limits.
func (p Argon2Params) validate() error {
	switch {
	case p.Memory < 8*uint32(p.Parallelism) || p.Memory > maxArgon2Memory:
		return fmt.Errorf("memory %d KiB out of range [8*p, %d]", p.Memory, maxArgon2Memory)
	case p.Iterations < 1 || p.Iterations > maxArgon2Iterations:
		return fmt.Errorf("iterations %d out of range [1, %d]", p.Iterations, maxArgon2Iterations)
	case p.Parallelism < 1 || p.Parallelism > maxArgon2Parallelism:
		return fmt.Errorf("parallelism %d out of range [1, %d]", p.Parallelism, maxArgon2Parallelism)
	case p.SaltLength < minArgon2Bytes || p.SaltLength > maxArgon2Bytes:
		return fmt.Errorf("salt length %d out of range [%d, %d]", p.SaltLength, minArgon2Bytes, maxArgon2Bytes)
	case p.KeyLength < minArgon2Bytes || p.KeyLength > maxArgon2Bytes:
		return fmt.Errorf("key length %d out of range [%d, %d]", p.KeyLength, minArgon2Bytes, maxArgon2Bytes)
	}
	return nil
}

// phcEncoding is the PHC base64: standard alphabet, no padding, and no
// non-canonical trailing bits.
var phcEncoding = base64.RawStdEncoding.Strict()

// errBadHash is returned for every malformed or out-of-limits PHC string.
// It does not say which part is wrong, so the error is safe to log.
var errBadHash = fmt.Errorf("security: verify password: malformed hash: %w", domain.ErrInvalidCredentials)

// HashPassword gera o hash Argon2id da senha com sal aleatório
// (crypto/rand) e o codifica no formato PHC:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<sal-base64>$<hash-base64>
//
// The password is normalized to NFKC first (D-14). Argon2id runs only after
// a slot of the hash semaphore is taken (D-19); if ctx ends while waiting,
// the context error is returned.
func (s *Security) HashPassword(ctx context.Context, password string) (string, error) {
	p := s.argon2
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("security: hash password: salt: %w", err)
	}

	key, err := s.deriveKey(ctx, normalizePassword(password), salt, p)
	if err != nil {
		return "", fmt.Errorf("security: hash password: %w", err)
	}
	return fmt.Sprintf("$argon2id$v=%d$%s$%s$%s",
		argon2.Version, formatParams(p), phcEncoding.EncodeToString(salt), phcEncoding.EncodeToString(key)), nil
}

// VerifyPassword compara a senha com um hash PHC em tempo constante.
// Retorna nil somente quando a senha confere; qualquer outro caso
// (incluindo hash malformado) retorna erro que encapsula
// domain.ErrInvalidCredentials.
//
// The exception is ctx ending while waiting for a semaphore slot (D-19):
// that returns the context error, which is a server failure (500), not a
// credential failure. The parameters come from the hash and must be within
// the D-15 limits.
func (s *Security) VerifyPassword(ctx context.Context, encodedHash, password string) error {
	p, salt, want, ok := parsePHC(encodedHash)
	if !ok {
		return errBadHash
	}

	got, err := s.deriveKey(ctx, normalizePassword(password), salt, p)
	if err != nil {
		return fmt.Errorf("security: verify password: %w", err)
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return fmt.Errorf("security: verify password: %w", domain.ErrInvalidCredentials)
	}
	return nil
}

// deriveKey runs Argon2id while holding a slot of the hash semaphore.
func (s *Security) deriveKey(ctx context.Context, password string, salt []byte, p Argon2Params) ([]byte, error) {
	if err := s.acquireHashSlot(ctx); err != nil {
		return nil, err
	}
	defer s.releaseHashSlot()
	return argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength), nil
}

// acquireHashSlot waits for a free slot or for ctx to end. A context that
// has already ended never takes a slot.
func (s *Security) acquireHashSlot(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("wait for hash slot: %w", err)
	}
	select {
	case s.hashSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for hash slot: %w", ctx.Err())
	}
}

func (s *Security) releaseHashSlot() {
	<-s.hashSlots
}

func formatParams(p Argon2Params) string {
	return fmt.Sprintf("m=%d,t=%d,p=%d", p.Memory, p.Iterations, p.Parallelism)
}

// parsePHC decodes $argon2id$v=19$m=<m>,t=<t>,p=<p>$<salt>$<key>. Only the
// canonical form is accepted: the exact field order, decimal numbers
// without leading zeros or signs, and strict unpadded base64.
func parsePHC(encoded string) (p Argon2Params, salt, key []byte, ok bool) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" ||
		parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return p, nil, nil, false
	}

	fields := strings.Split(parts[3], ",")
	if len(fields) != 3 {
		return p, nil, nil, false
	}
	m, okM := parseParam(fields[0], "m=", maxArgon2Memory)
	t, okT := parseParam(fields[1], "t=", maxArgon2Iterations)
	par, okP := parseParam(fields[2], "p=", maxArgon2Parallelism)
	if !okM || !okT || !okP {
		return p, nil, nil, false
	}

	var err error
	if salt, err = phcEncoding.DecodeString(parts[4]); err != nil {
		return p, nil, nil, false
	}
	if key, err = phcEncoding.DecodeString(parts[5]); err != nil {
		return p, nil, nil, false
	}

	// Bounds checked here, next to the conversions below.
	saltLen, keyLen := len(salt), len(key)
	if par > maxArgon2Parallelism ||
		saltLen < minArgon2Bytes || saltLen > maxArgon2Bytes ||
		keyLen < minArgon2Bytes || keyLen > maxArgon2Bytes {
		return p, nil, nil, false
	}
	p = Argon2Params{
		Memory:      m,
		Iterations:  t,
		Parallelism: uint8(par),
		SaltLength:  uint32(saltLen),
		KeyLength:   uint32(keyLen),
	}
	if p.validate() != nil || formatParams(p) != parts[3] {
		return p, nil, nil, false
	}
	return p, salt, key, true
}

// parseParam parses "<prefix><n>" with 0 < n ≤ limit.
func parseParam(field, prefix string, limit uint32) (uint32, bool) {
	digits, found := strings.CutPrefix(field, prefix)
	if !found || digits == "" || len(digits) > 10 {
		return 0, false
	}
	n, err := strconv.ParseUint(digits, 10, 32)
	if err != nil || n == 0 || n > uint64(limit) {
		return 0, false
	}
	return uint32(n), true
}
