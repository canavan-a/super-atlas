package friends

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/go-chi/chi/v5"
)

// LarpToggler is implemented by the larp package. Declared here (rather than
// importing larp directly) to avoid an import cycle, since larp depends on
// this package to create fake friendships.
type LarpToggler interface {
	Toggle(ctx context.Context, ownerID string) (active bool, err error)
}

type Handlers struct {
	service *Service
	// larp is nil unless the server runs in dev mode.
	larp LarpToggler
	// inviteURL turns an invite code into a shareable link.
	inviteURL func(r *http.Request, code string) string
}

func NewHandlers(service *Service, larp LarpToggler, inviteURL func(r *http.Request, code string) string) *Handlers {
	return &Handlers{service: service, larp: larp, inviteURL: inviteURL}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type sendRequestBody struct {
	Email string `json:"email"`
}

func (h *Handlers) SendRequest(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body sendRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Email) == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	if h.larp != nil && strings.EqualFold(strings.TrimSpace(body.Email), "larp") {
		active, err := h.larp.Toggle(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not toggle larp")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"larp": active})
		return
	}

	f, err := h.service.SendRequest(r.Context(), userID, body.Email)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			writeError(w, http.StatusNotFound, "nobody has signed up with that email yet; send them your invite link instead")
		case errors.Is(err, ErrSelfRequest):
			writeError(w, http.StatusBadRequest, "that's your own email")
		case errors.Is(err, ErrAlreadyExists):
			writeError(w, http.StatusConflict, "you're already friends, or a request is already pending")
		default:
			writeError(w, http.StatusInternalServerError, "could not send friend request")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"friendshipId": f.ID, "status": f.Status})
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	views, err := h.service.List(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list friends")
		return
	}
	writeJSON(w, http.StatusOK, views)
}

func (h *Handlers) Accept(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.service.Accept(r.Context(), chi.URLParam(r, "id"), userID); err != nil {
		handleActionErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Remove(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.service.Remove(r.Context(), chi.URLParam(r, "id"), userID); err != nil {
		handleActionErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type inviteResponse struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

// Invite returns the caller's personal invite link.
func (h *Handlers) Invite(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	code, err := h.service.InviteCode(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load invite link")
		return
	}
	writeJSON(w, http.StatusOK, inviteResponse{Code: code, URL: h.inviteURL(r, code)})
}

// ResetInvite replaces the caller's invite link; the old one stops working.
func (h *Handlers) ResetInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	code, err := h.service.ResetInviteCode(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset invite link")
		return
	}
	writeJSON(w, http.StatusOK, inviteResponse{Code: code, URL: h.inviteURL(r, code)})
}

// InvitePreview says whose invite a code is, so the app can ask "Add <name>?".
func (h *Handlers) InvitePreview(w http.ResponseWriter, r *http.Request) {
	name, err := h.service.InviteOwner(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		handleActionErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"displayName": name})
}

func (h *Handlers) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	name, err := h.service.AcceptInvite(r.Context(), userID, chi.URLParam(r, "code"))
	if err != nil {
		handleActionErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"displayName": name})
}

func handleActionErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "friendship not found")
	case errors.Is(err, ErrInvalidInvite):
		writeError(w, http.StatusNotFound, "this invite link doesn't work anymore")
	case errors.Is(err, ErrSelfRequest):
		writeError(w, http.StatusBadRequest, "that's your own invite link")
	case errors.Is(err, ErrNotYourRequest):
		writeError(w, http.StatusForbidden, "not your friendship")
	case errors.Is(err, ErrNotPending):
		writeError(w, http.StatusConflict, "friendship is not pending")
	default:
		writeError(w, http.StatusInternalServerError, "action failed")
	}
}
