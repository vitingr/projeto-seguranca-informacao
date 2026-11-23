package web

import (
	"log/slog"
	"net/http"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
)

type auditActorResponse struct {
	UserID domain.UserID `json:"user_id"`
	// Login is omitted when the user no longer exists.
	Login string `json:"login,omitempty"`
}

type auditTargetResponse struct {
	Type domain.AuditTargetType `json:"type"`
	ID   int64                  `json:"id"`
	// Login is set only for a user target that still exists.
	Login string `json:"login,omitempty"`
}

type auditEventResponse struct {
	ID        domain.AuditEventID   `json:"id"`
	Event     domain.AuditEventType `json:"event"`
	Actor     *auditActorResponse   `json:"actor"`
	Target    *auditTargetResponse  `json:"target"`
	IP        *string               `json:"ip"`
	CreatedAt time.Time             `json:"created_at"`
}

type auditPageResponse struct {
	Items    []auditEventResponse `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	HasNext  bool                 `json:"has_next"`
}

func toAuditEventResponse(ev *domain.AuditEvent) auditEventResponse {
	out := auditEventResponse{ID: ev.ID, Event: ev.Type, CreatedAt: ev.CreatedAt.UTC()}
	if ev.ActorID != 0 {
		out.Actor = &auditActorResponse{UserID: ev.ActorID, Login: ev.ActorLogin}
	}
	if ev.TargetType != "" {
		out.Target = &auditTargetResponse{Type: ev.TargetType, ID: ev.TargetID, Login: ev.TargetLogin}
	}
	if ev.IP.IsValid() {
		ip := ev.IP.String()
		out.IP = &ip
	}
	return out
}

func (h *handlers) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	typ, err := h.sec.ParseAuditType(query.Get("type"))
	if err != nil {
		h.badRequest(w, r, err)
		return
	}
	page, err := h.sec.ParsePage(query.Get("page"))
	if err != nil {
		h.badRequest(w, r, err)
		return
	}
	result, err := h.audit.List(r.Context(), typ, page)
	if err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	items := make([]auditEventResponse, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toAuditEventResponse(&result.Items[i]))
	}
	writeJSON(w, http.StatusOK, auditPageResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		HasNext:  result.HasNext,
	})
}

// record appends ev to the audit log with the IP of the TCP peer (006 D-5).
// It runs after the outcome of the request (D-6): a failure is logged at
// ERROR, with the event and IDs only, and never changes the response,
// because the session or the user already exists.
func (h *handlers) record(r *http.Request, ev domain.AuditEvent) {
	if ip, ok := h.sec.ClientIP(r); ok {
		ev.IP = ip
	}
	if err := h.audit.Record(r.Context(), ev); err != nil {
		h.logger.ErrorContext(r.Context(), "audit: record event failed",
			slog.String("event", string(ev.Type)),
			slog.Int64("actor_user_id", int64(ev.ActorID)),
			slog.String("target_type", string(ev.TargetType)),
			slog.Int64("target_id", ev.TargetID),
			slog.Any("error", err),
		)
	}
}

// callerID is the user ID of the caller, or 0 for an anonymous request.
func callerID(r *http.Request) domain.UserID {
	if p, ok := security.PrincipalFrom(r.Context()); ok {
		return p.UserID
	}
	return 0
}
