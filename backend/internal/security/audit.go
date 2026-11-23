package security

import (
	"fmt"
	"net/http"
	"net/netip"

	"backend_si/internal/domain"
)

// ClientIP returns the address of the direct TCP peer of r, from
// r.RemoteAddr without the port (006 D-5). X-Forwarded-For, X-Real-IP and
// every other header are ignored on purpose: any browser can set them, and
// the Next rewrite passes them on unchanged, so a header value would let a
// caller write any IP into the audit log. ok is false when RemoteAddr does
// not parse; the event is then stored without an IP.
func (s *Security) ClientIP(r *http.Request) (netip.Addr, bool) {
	return clientIP(r.RemoteAddr)
}

func clientIP(remoteAddr string) (netip.Addr, bool) {
	ap, err := netip.ParseAddrPort(remoteAddr)
	if err != nil {
		return netip.Addr{}, false
	}
	// INET has no zone, and an IPv4-mapped IPv6 peer is an IPv4 client.
	return ap.Addr().WithZone("").Unmap(), true
}

// ParseAuditType reads the type filter of the audit log (006 D-8): empty is
// no filter (""), otherwise one of the five events. Anything else wraps
// domain.ErrInvalidInput (400).
func (s *Security) ParseAuditType(raw string) (domain.AuditEventType, error) {
	if raw == "" {
		return "", nil
	}
	t := domain.AuditEventType(raw)
	if !t.Valid() {
		return "", fmt.Errorf("security: parse audit type: %w", domain.ErrInvalidInput)
	}
	return t, nil
}
