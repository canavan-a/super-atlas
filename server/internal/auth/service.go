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
	ErrFull           = errors.New("the user cap is reached")
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

// Mailer sends the sign-in code out of band, and the new-account alert;
// satisfied by mail.Sender.
type Mailer interface {
	SendLoginCode(to, code string) error
	SendSignupAlert(to string, user *User, count, cap int) error
}

type Service struct {
	store  *Store
	mailer Mailer
	now    func() time.Time
	sends  *ratelimit.Limiter
	// The app store review account (SetReviewAccount): empty when off.
	reviewEmail string
	reviewCode  string
	// alertEmail hears about every new account (SetSignupAlert): empty
	// when off.
	alertEmail string
}

func NewService(store *Store, mailer Mailer) *Service {
	return &Service{store: store, mailer: mailer, now: time.Now, sends: ratelimit.New(codesPerHour, time.Hour)}
}

// SetReviewAccount turns on a sign-in for app store reviewers: email always
// signs in with the fixed code, nothing is emailed, and it can sign up even
// when the user cap is reached. Otherwise it's an ordinary account (friends
// can add it). Empty email or code leaves it off.
func (s *Service) SetReviewAccount(email, code string) {
	if email = NormalizeEmail(email); email == "" || code == "" {
		return
	}
	s.reviewEmail, s.reviewCode = email, code
}

// SetSignupAlert emails to whenever someone makes an account. Empty: off.
func (s *Service) SetSignupAlert(to string) {
	s.alertEmail = strings.TrimSpace(to)
}

func (s *Service) isReviewAccount(email string) bool {
	return s.reviewEmail != "" && email == s.reviewEmail
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// RequestCode emails a fresh 6-digit code to email. Anyone with an account
// gets one; anyone else only while there are fewer accounts than the user cap
// (ErrFull otherwise, and nothing is sent), since the same code signs up.
func (s *Service) RequestCode(ctx context.Context, email string) error {
	email = NormalizeEmail(email)
	if !strings.Contains(email, "@") || len(email) > 254 {
		return ErrInvalidEmail
	}
	if _, err := s.store.GetUserByEmail(ctx, email); errors.Is(err, ErrNotFound) {
		if err := s.checkRoom(ctx, email); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if s.isReviewAccount(email) {
		return nil // its code is fixed, so there's nothing to send
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
	// The review account has no stored code to check or use up.
	var lc *LoginCode
	if s.isReviewAccount(email) {
		if subtle.ConstantTimeCompare([]byte(s.reviewCode), []byte(strings.TrimSpace(code))) != 1 {
			return nil, "", ErrInvalidCode
		}
	} else {
		var err error
		lc, err = s.store.LatestLoginCode(ctx, email)
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
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		// The last place may have gone since the code was sent.
		if err := s.checkRoom(ctx, email); err != nil {
			return nil, "", err
		}
		if displayName == "" {
			return nil, "", ErrNeedsName
		}
		var created bool
		user, created, err = s.createUser(ctx, email, displayName)
		if err == nil && created {
			s.alertSignup(ctx, user)
		}
	}
	if err != nil {
		return nil, "", err
	}

	if lc != nil {
		if err := s.store.MarkLoginCodeUsed(ctx, lc.ID); err != nil {
			return nil, "", err
		}
	}
	token, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// checkRoom returns ErrFull if email (which has no account) can't make one
// because the user cap is reached.
func (s *Service) checkRoom(ctx context.Context, email string) error {
	if s.isReviewAccount(email) {
		return nil
	}
	n, err := s.store.CountUsers(ctx)
	if err != nil {
		return err
	}
	limit, err := s.store.UserCap(ctx)
	if err != nil {
		return err
	}
	if n >= limit {
		return ErrFull
	}
	return nil
}

// createUser makes the account, reporting whether this call made it.
func (s *Service) createUser(ctx context.Context, email, displayName string) (*User, bool, error) {
	id := uuid.NewString()
	if err := s.store.CreateUser(ctx, id, email, displayName); err != nil {
		// Two verifies for the same new email racing: the loser's insert
		// hits the UNIQUE constraint, and the winner's account is the one.
		if u, getErr := s.store.GetUserByEmail(ctx, email); getErr == nil {
			return u, false, nil
		}
		return nil, false, err
	}
	u, err := s.store.GetUserByID(ctx, id)
	return u, err == nil, err
}

// alertSignup tells the alert email about user's new account, in the
// background so a slow mail server doesn't hold up their sign-in.
func (s *Service) alertSignup(ctx context.Context, user *User) {
	if s.alertEmail == "" {
		return
	}
	count, err := s.store.CountUsers(ctx)
	if err != nil {
		log.Printf("signup alert: %v", err)
		return
	}
	limit, err := s.store.UserCap(ctx)
	if err != nil {
		log.Printf("signup alert: %v", err)
		return
	}
	go func() {
		if err := s.mailer.SendSignupAlert(s.alertEmail, user, count, limit); err != nil {
			log.Printf("signup alert for %s: %v", user.Email, err)
		}
	}()
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

// DeleteAccount deletes userID's account and all its data, signing out every
// device it was on.
func (s *Service) DeleteAccount(ctx context.Context, userID string) error {
	return s.store.DeleteUser(ctx, userID)
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
