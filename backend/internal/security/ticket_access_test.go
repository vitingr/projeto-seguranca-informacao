package security

import (
	"testing"

	"backend_si/internal/domain"
)

func TestParseTicketID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		want   domain.TicketID
		wantOK bool
	}{
		{"one", "1", 1, true},
		{"two digits", "42", 42, true},
		{"largest int64", "9223372036854775807", 9223372036854775807, true},

		{"zero", "0", 0, false},
		{"negative", "-1", 0, false},
		{"plus sign", "+1", 0, false},
		{"leading zero", "01", 0, false},
		{"only zeros", "00", 0, false},
		{"exponent", "1e3", 0, false},
		{"decimal", "1.0", 0, false},
		{"hex", "0x1", 0, false},
		{"letters", "abc", 0, false},
		{"leading space", " 1", 0, false},
		{"trailing space", "1 ", 0, false},
		{"trailing newline", "1\n", 0, false},
		{"empty", "", 0, false},
		{"one past int64", "9223372036854775808", 0, false},
		{"twenty digits", "10000000000000000000", 0, false},
		{"percent-encoded digit", "%31", 0, false},
		{"arabic-indic digit", "١", 0, false},
		{"fullwidth digit", "１", 0, false},
		{"digit then fullwidth digit", "1２", 0, false},
		{"underscore", "1_000", 0, false},
		{"path", "1/2", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseTicketID(tt.raw)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ParseTicketID(%q) = %d, %v; want %d, %v", tt.raw, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCanAccessTicket(t *testing.T) {
	t.Parallel()

	const owner, other domain.UserID = 10, 11
	tests := []struct {
		name      string
		principal Principal
		author    domain.UserID
		want      bool
	}{
		{"admin on any ticket", Principal{UserID: 1, Role: domain.RoleAdmin}, owner, true},
		{"admin on a ticket of another admin id", Principal{UserID: owner, Role: domain.RoleAdmin}, other, true},
		{"comum author", Principal{UserID: owner, Role: domain.RoleComum}, owner, true},

		// BR-1: for a comum user, every other ticket does not exist.
		{"other comum user", Principal{UserID: other, Role: domain.RoleComum}, owner, false},

		// Fail closed.
		{"no principal", Principal{}, owner, false},
		{"zero user id, comum", Principal{Role: domain.RoleComum}, owner, false},
		{"zero user id, admin", Principal{Role: domain.RoleAdmin}, owner, false},
		{"negative user id", Principal{UserID: -1, Role: domain.RoleAdmin}, owner, false},
		{"zero author, admin", Principal{UserID: 1, Role: domain.RoleAdmin}, 0, false},
		{"zero author and zero user, comum", Principal{Role: domain.RoleComum}, 0, false},
		{"negative author", Principal{UserID: 1, Role: domain.RoleAdmin}, -1, false},
		{"unknown role", Principal{UserID: owner, Role: "root"}, owner, false},
		{"empty role", Principal{UserID: owner}, owner, false},
		{"role in another case", Principal{UserID: owner, Role: "Admin"}, owner, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CanAccessTicket(tt.principal, tt.author); got != tt.want {
				t.Errorf("CanAccessTicket(%+v, %d) = %v, want %v", tt.principal, tt.author, got, tt.want)
			}
		})
	}
}
