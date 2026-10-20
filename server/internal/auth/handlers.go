package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/ratelimit"
)

// GhostModeGetter reports a user's ghost-mode state for /me. Implemented by
// the location service; declared here to avoid an import cycle.
type GhostModeGetter interface {
	GhostMode(ctx context.Context, userID string) (bool, error)
}

type Handlers struct {
	service   *Service
	ghostMode GhostModeGetter
	// Per email, however many IPs ask: one code a minute and ten a day, and
	// ten guesses an hour across all its codes (each code also allows only
	// five), so a code can't be guessed by asking for new ones.
	codesPerEmail      *ratelimit.Limiter
	codesPerEmailDaily *ratelimit.Limiter
	guessesPerEmail    *ratelimit.Limiter
}

func NewHandlers(service *Service, ghostMode GhostModeGetter) *Handlers {
	return &Handlers{
		service:            service,
		ghostMode:          ghostMode,
		codesPerEmail:      ratelimit.New(1, time.Minute),
		codesPerEmailDaily: ratelimit.New(10, 24*time.Hour),
		guessesPerEmail:    ratelimit.New(10, time.Hour),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

func toUserResponse(u *User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName}
}

const notAllowedMessage = "this email isn't on the Super Atlas guest list yet, ask whoever runs it to add you"

type codeRequest struct {
	Email string `json:"email"`
}

// Code emails a sign-in code. The response is the same whether or not the
// email has an account.
func (h *Handlers) Code(w http.ResponseWriter, r *http.Request) {
	var req codeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	email := NormalizeEmail(req.Email)
	if !h.codesPerEmail.Allow(email) {
		writeError(w, http.StatusTooManyRequests, "wait a minute before asking for another code")
		return
	}
	if !h.codesPerEmailDaily.Allow(email) {
		writeError(w, http.StatusTooManyRequests, "too many codes today, try again tomorrow")
		return
	}
	if err := h.service.RequestCode(r.Context(), req.Email); err != nil {
		if errors.Is(err, ErrInvalidEmail) {
			writeError(w, http.StatusBadRequest, "enter a valid email")
			return
		}
		if errors.Is(err, ErrNotAllowed) {
			writeError(w, http.StatusForbidden, notAllowedMessage)
			return
		}
		if errors.Is(err, ErrTooManyCodes) {
			writeError(w, http.StatusTooManyRequests, "too many sign-ins right now, try again in a while")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not send code")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type verifyRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
}

type verifyResponse struct {
	User  userResponse `json:"user"`
	Token string       `json:"token"`
}

func (h *Handlers) Verify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !h.guessesPerEmail.Allow(NormalizeEmail(req.Email)) {
		writeError(w, http.StatusTooManyRequests, "too many tries, wait a while and ask for a new code")
		return
	}
	user, token, err := h.service.Verify(r.Context(), req.Email, req.Code, req.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCode):
			writeError(w, http.StatusUnauthorized, "that code is wrong or expired")
		case errors.Is(err, ErrNotAllowed):
			writeError(w, http.StatusForbidden, notAllowedMessage)
		case errors.Is(err, ErrNeedsName):
			writeJSON(w, http.StatusConflict, map[string]any{"error": "new account needs a name", "needsName": true})
		default:
			writeError(w, http.StatusInternalServerError, "could not sign in")
		}
		return
	}
	writeJSON(w, http.StatusOK, verifyResponse{User: toUserResponse(user), Token: token})
}

type meResponse struct {
	User      userResponse `json:"user"`
	GhostMode bool         `json:"ghostMode"`
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	user, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "signed out")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load account")
		return
	}
	ghost, err := h.ghostMode.GhostMode(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load account")
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: toUserResponse(user), GhostMode: ghost})
}

// Logout ends the session the request was made with.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Logout(r.Context(), tokenFromContext(r.Context())); err != nil {
		writeError(w, http.StatusInternalServerError, "could not sign out")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
