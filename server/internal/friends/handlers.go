package friends

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/canavan-a/open-location/server/internal/auth"
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
	larp    LarpToggler
}

func NewHandlers(service *Service, larp LarpToggler) *Handlers {
	return &Handlers{service: service, larp: larp}
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	if strings.EqualFold(strings.TrimSpace(body.Email), "larp") {
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
			writeError(w, http.StatusNotFound, "no user with that email")
		case errors.Is(err, ErrSelfRequest):
			writeError(w, http.StatusBadRequest, "cannot friend yourself")
		case errors.Is(err, ErrAlreadyExists):
			writeError(w, http.StatusConflict, "friendship already exists")
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
	id := chi.URLParam(r, "id")
	if err := h.service.Accept(r.Context(), id, userID); err != nil {
		handleActionErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Remove(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.service.Remove(r.Context(), id, userID); err != nil {
		handleActionErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type setSharingBody struct {
	Sharing bool `json:"sharing"`
}

func (h *Handlers) SetSharing(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var body setSharingBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.SetSharing(r.Context(), id, userID, body.Sharing); err != nil {
		handleActionErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetName(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	otherID := chi.URLParam(r, "userId")
	displayName, err := h.service.GetDisplayName(r.Context(), userID, otherID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not look up name")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"userId": otherID, "displayName": displayName})
}

func handleActionErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "friendship not found")
	case errors.Is(err, ErrNotYourRequest):
		writeError(w, http.StatusForbidden, "not your friendship")
	case errors.Is(err, ErrNotPending):
		writeError(w, http.StatusConflict, "friendship is not pending")
	default:
		writeError(w, http.StatusInternalServerError, "action failed")
	}
}
