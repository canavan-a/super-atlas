package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/ratelimit"
	"github.com/google/uuid"
)

var (
	ErrInvalidEmail   = errors.New("invalid email")
	ErrInvalidCode    = errors.New("invalid or expired code")
	ErrNeedsName      = errors.New("display name required for a new account")
	ErrInvalidSession = errors.New("invalid or expired session")
	ErrNotAllowed     = errors.New("email isn't on the allowlist")
	ErrTooManyCodes   = errors.New("too many sign-in codes sent")
)

const (
	codeTTL         = 10 * time.Minute
	maxCodeAttempts = 5
	// A session is only ever signed out after this long without being used.
	sessionIdleTTL = 90 * 24 * time.Hour
	// last_used_at/expires_at are rewritten at most this often per session,
	// so ordinary requests don't each cost a database write.
	sessionTouchEvery = time.Hour
	// The most sign-in codes sent in an hour, to everyone together: a cap on
	// what the server can make the mail account send, well under Gmail's
	// daily limit.
	codesPerHour = 100
)

// Mailer sends the sign-in code out of band; satisfied by mail.Sender.
type Mailer interface {
	SendLoginCode(to, code string) error
}

type Service struct {
	store  *Store
	mailer Mailer
	now    func() time.Time
	sends  *ratelimit.Limiter
}

func NewService(store *Store, mailer Mailer) *Service {
	return &Service{store: store, mailer: mailer, now: time.Now, sends: ratelimit.New(codesPerHour, time.Hour)}
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// RequestCode emails a fresh 6-digit code to email, if email is on the
// allowlist (ErrNotAllowed otherwise, and nothing is sent). It behaves the
// same whether or not an account exists, since the same code also signs up.
func (s *Service) RequestCode(ctx context.Context, email string) error {
	email = NormalizeEmail(email)
	if !strings.Contains(email, "@") || len(email) > 254 {
		return ErrInvalidEmail
	}
	if err := s.checkAllowed(ctx, email); err != nil {
		return err
	}
	if !s.sends.Allow("all") {
		return ErrTooManyCodes
	}

	code, err := generateCode()
	if err != nil {
		return err
	}
	if err := s.store.CreateLoginCode(ctx, uuid.NewString(), email, HashToken(code), s.now().Add(codeTTL)); err != nil {
		return err
	}
	return s.mailer.SendLoginCode(email, code)
}

// Verify checks code against the latest one sent to email and, if it
// matches, signs in (creating the account when there isn't one yet) and
// returns a new session token. A new account needs displayName; without one
// it returns ErrNeedsName and leaves the code usable for the retry.
func (s *Service) Verify(ctx context.Context, email, code, displayName string) (*User, string, error) {
	email = NormalizeEmail(email)
	displayName = strings.TrimSpace(displayName)
	// Someone taken off the allowlist can't use a code they already had.
	if err := s.checkAllowed(ctx, email); err != nil {
		return nil, "", err
	}

	lc, err := s.store.LatestLoginCode(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, "", ErrInvalidCode
		}
		return nil, "", err
	}
	if lc.UsedAt != nil || s.now().After(lc.ExpiresAt) || lc.Attempts >= maxCodeAttempts {
		return nil, "", ErrInvalidCode
	}
	if subtle.ConstantTimeCompare([]byte(lc.CodeHash), []byte(HashToken(strings.TrimSpace(code)))) != 1 {
		if err := s.store.IncrementLoginCodeAttempts(ctx, lc.ID); err != nil {
			return nil, "", err
		}
		return nil, "", ErrInvalidCode
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		if displayName == "" {
			return nil, "", ErrNeedsName
		}
		user, err = s.createUser(ctx, email, displayName)
	}
	if err != nil {
		return nil, "", err
	}

	if err := s.store.MarkLoginCodeUsed(ctx, lc.ID); err != nil {
		return nil, "", err
	}
	token, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *Service) checkAllowed(ctx context.Context, email string) error {
	ok, err := s.store.IsAllowed(ctx, email)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotAllowed
	}
	return nil
}

func (s *Service) createUser(ctx context.Context, email, displayName string) (*User, error) {
	id := uuid.NewString()
	if err := s.store.CreateUser(ctx, id, email, displayName); err != nil {
		// Two verifies for the same new email racing: the loser's insert
		// hits the UNIQUE constraint, and the winner's account is the one.
		if u, getErr := s.store.GetUserByEmail(ctx, email); getErr == nil {
			return u, nil
		}
		return nil, err
	}
	return s.store.GetUserByID(ctx, id)
}

func (s *Service) createSession(ctx context.Context, userID string) (string, error) {
	raw, hash, err := GenerateOpaqueToken()
	if err != nil {
		return "", err
	}
	now := s.now()
	if err := s.store.CreateSession(ctx, uuid.NewString(), userID, hash, now, now.Add(sessionIdleTTL)); err != nil {
		return "", err
	}
	return raw, nil
}

// Authenticate resolves a session token to its user, sliding the session's
// expiry forward as it's used.
func (s *Service) Authenticate(ctx context.Context, token string) (string, error) {
	sess, err := s.session(ctx, token)
	if err != nil {
		return "", err
	}
	return sess.UserID, nil
}

func (s *Service) session(ctx context.Context, token string) (*Session, error) {
	sess, err := s.store.GetSessionByHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidSession
		}
		return nil, err
	}
	now := s.now()
	if now.After(sess.ExpiresAt) {
		_ = s.store.DeleteSession(ctx, sess.ID)
		return nil, ErrInvalidSession
	}
	if now.Sub(sess.LastUsedAt) > sessionTouchEvery {
		if err := s.store.TouchSession(ctx, sess.ID, now, now.Add(sessionIdleTTL)); err != nil {
			return nil, err
		}
	}
	return sess, nil
}

// Logout ends the session for token. Unknown tokens are a no-op.
func (s *Service) Logout(ctx context.Context, token string) error {
	sess, err := s.store.GetSessionByHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	return s.store.DeleteSession(ctx, sess.ID)
}

func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
	return s.store.GetUserByID(ctx, id)
}

// RunJanitor deletes expired codes and sessions every hour until ctx ends.
func (s *Service) RunJanitor(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		now := s.now()
		if err := s.store.DeleteExpiredLoginCodes(ctx, now); err != nil {
			log.Printf("janitor: login codes: %v", err)
		}
		if err := s.store.DeleteExpiredSessions(ctx, now); err != nil {
			log.Printf("janitor: sessions: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
