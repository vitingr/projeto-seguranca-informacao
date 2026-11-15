package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend_si/internal/domain"
)

// TestAuthorize is the D-1 matrix: zero policy → 403, public → allow,
// anonymous → 401, role in the policy → allow, any other role → 403.
func TestAuthorize(t *testing.T) {
	t.Parallel()

	admin := &Principal{UserID: 1, Role: domain.RoleAdmin}
	comum := &Principal{UserID: 2, Role: domain.RoleComum}
	badRole := &Principal{UserID: 3, Role: "root"} // not a valid principal
	noUser := &Principal{Role: domain.RoleAdmin}   // not a valid principal
	policies := map[string]Policy{
		"zero":         {},
		"empty roles":  RequireRole(),
		"public":       Public(),
		"admin":        RequireRole(domain.RoleAdmin),
		"comum":        RequireRole(domain.RoleComum),
		"any user":     RequireRole(domain.RoleAdmin, domain.RoleComum),
		"invalid only": RequireRole("root"),
	}
	callers := map[string]*Principal{"anonymous": nil, "admin": admin, "comum": comum, "bad role": badRole, "no user": noUser}

	// want[policy][caller] = status (200 = handler ran).
	want := map[string]map[string]int{
		"zero":         {"anonymous": 403, "admin": 403, "comum": 403, "bad role": 403, "no user": 403},
		"empty roles":  {"anonymous": 403, "admin": 403, "comum": 403, "bad role": 403, "no user": 403},
		"invalid only": {"anonymous": 403, "admin": 403, "comum": 403, "bad role": 403, "no user": 403},
		"public":       {"anonymous": 200, "admin": 200, "comum": 200, "bad role": 200, "no user": 200},
		"admin":        {"anonymous": 401, "admin": 200, "comum": 403, "bad role": 401, "no user": 401},
		"comum":        {"anonymous": 401, "admin": 403, "comum": 200, "bad role": 401, "no user": 401},
		"any user":     {"anonymous": 401, "admin": 200, "comum": 200, "bad role": 401, "no user": 401},
	}
	if len(want) != len(policies) {
		t.Fatal("the expectation table does not cover every policy")
	}

	for pname, policy := range policies {
		for cname, caller := range callers {
			t.Run(pname+"/"+cname, func(t *testing.T) {
				t.Parallel()
				s := newTestSecurity(t, nil)
				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", http.NoBody)
				if caller != nil {
					req = req.WithContext(context.WithValue(req.Context(), principalKey{}, *caller))
				}
				var called bool
				rr := httptest.NewRecorder()
				s.Authorize(policy)(okHandler(&called)).ServeHTTP(rr, req)

				wantStatus := want[pname][cname]
				if called != (wantStatus == http.StatusOK) {
					t.Errorf("handler ran = %v, want status %d", called, wantStatus)
				}
				if rr.Code != wantStatus {
					t.Errorf("status = %d, want %d", rr.Code, wantStatus)
				}
			})
		}
	}
}

// TestAuthorizeWithSession runs Authorize behind the real Session
// middleware: a logged-in admin passes, the same cookie after logout gets
// 401.
func TestAuthorizeWithSession(t *testing.T) {
	t.Parallel()

	store := newMemStore()
	s := newSessionSecurity(t, store, newClock(), nil)
	token := login(t, s, 1, domain.RoleAdmin)

	do := func() int {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/administradores", http.NoBody)
		req.AddCookie(reqCookie(token))
		var called bool
		rr := httptest.NewRecorder()
		s.Session(s.Authorize(RequireRole(domain.RoleAdmin))(okHandler(&called))).ServeHTTP(rr, req)
		return rr.Code
	}
	if code := do(); code != http.StatusOK {
		t.Fatalf("logged-in admin: status = %d, want 200", code)
	}
	if err := store.Delete(t.Context(), hashToken(token)); err != nil {
		t.Fatal(err)
	}
	if code := do(); code != http.StatusUnauthorized {
		t.Errorf("after the session is deleted: status = %d, want 401", code)
	}
}
