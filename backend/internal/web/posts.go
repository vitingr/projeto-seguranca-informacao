package web

import (
	"errors"
	"net/http"
	"time"

	"backend_si/internal/domain"
	"backend_si/internal/security"
	"backend_si/internal/service"
)

const msgPostNaoEncontrado = "Post não encontrado."

type postRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type postAuthorResponse struct {
	UserID domain.UserID `json:"user_id"`
	Login  string        `json:"login"`
}

type postResponse struct {
	ID        domain.PostID      `json:"id"`
	Title     string             `json:"title"`
	Body      string             `json:"body"`
	Author    postAuthorResponse `json:"author"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Mine      bool               `json:"mine"`
}

type postPageResponse struct {
	Items    []postResponse `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	HasNext  bool           `json:"has_next"`
}

// toPostResponse sets mine from the caller of the request (005).
func toPostResponse(p domain.Post, caller domain.UserID) postResponse {
	return postResponse{
		ID:        p.ID,
		Title:     p.Title,
		Body:      p.Body,
		Author:    postAuthorResponse{UserID: p.AuthorID, Login: p.AuthorLogin},
		CreatedAt: p.CreatedAt.UTC(),
		UpdatedAt: p.UpdatedAt.UTC(),
		Mine:      p.AuthorID == caller,
	}
}

func (h *handlers) listPosts(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}
	page, err := h.sec.ParsePage(r.URL.Query().Get("page"))
	if err != nil {
		h.badRequest(w, r, err)
		return
	}
	result, err := h.posts.List(r.Context(), page)
	if err != nil {
		h.sec.RespondError(w, r, err)
		return
	}
	items := make([]postResponse, 0, len(result.Items))
	for _, post := range result.Items {
		items = append(items, toPostResponse(post, p.UserID))
	}
	writeJSON(w, http.StatusOK, postPageResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		HasNext:  result.HasNext,
	})
}

func (h *handlers) createPost(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}
	var req postRequest
	if err := h.sec.DecodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, err)
		return
	}
	post, err := h.posts.Create(r.Context(), p.UserID, service.PostInput{Title: req.Title, Body: req.Body})
	if err != nil {
		h.postError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPostResponse(post, p.UserID))
}

func (h *handlers) updatePost(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}
	id, err := h.sec.ParsePostID(r.PathValue("id"))
	if err != nil {
		h.postError(w, r, err)
		return
	}
	var req postRequest
	if err = h.sec.DecodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, err)
		return
	}
	post, err := h.posts.Update(r.Context(), p.UserID, id, service.PostInput{Title: req.Title, Body: req.Body})
	if err != nil {
		h.postError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPostResponse(post, p.UserID))
}

func (h *handlers) deletePost(w http.ResponseWriter, r *http.Request) {
	p, ok := security.PrincipalFrom(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized)
		return
	}
	id, err := h.sec.ParsePostID(r.PathValue("id"))
	if err != nil {
		h.postError(w, r, err)
		return
	}
	if err := h.posts.Delete(r.Context(), p.UserID, id); err != nil {
		h.postError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// postError maps the errors of the post operations: invalid fields → 422,
// a missing post or another user's post → 404 with the same body (D-3).
func (h *handlers) postError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeErrorMessage(w, r, http.StatusUnprocessableEntity, msgDadosInvalidos)
	case errors.Is(err, domain.ErrNotFound):
		writeErrorMessage(w, r, http.StatusNotFound, msgPostNaoEncontrado)
	default:
		h.sec.RespondError(w, r, err)
	}
}
