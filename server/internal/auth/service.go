package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEmailInUse         = errors.New("email already in use")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrRefreshExpired     = errors.New("refresh token expired or revoked")
)

// Mailer sends the reset code out of band; satisfied by mail.Sender.
type Mailer interface {
	SendPasswordResetCode(to, code string) error
}

type Service struct {
	store           *Store
	accessSecret    []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	mailer          Mailer
}

func NewService(store *Store, accessSecret []byte, accessTTL, refreshTTL time.Duration, mailer Mailer) *Service {
	return &Service{
		store:           store,
		accessSecret:    accessSecret,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		mailer:          mailer,
	}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *Service) Signup(ctx context.Context, email, password, displayName string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := s.store.GetUserByEmail(ctx, email); err == nil {
		return nil, ErrEmailInUse
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	id := uuid.NewString()
	if err := s.store.CreateUser(ctx, id, email, hash, displayName); err != nil {
		return nil, err
	}
	return s.store.GetUserByID(ctx, id)
}

func (s *Service) Login(ctx context.Context, email, password string) (*User, *TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !VerifyPassword(user.PasswordHash, password) {
		return nil, nil, ErrInvalidCredentials
	}

	pair, err := s.issueTokenPair(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	hash := hashToken(rawRefreshToken)
	rt, err := s.store.GetRefreshToken(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrRefreshExpired
		}
		return nil, err
	}
	if rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		return nil, ErrRefreshExpired
	}

	// Rotate: revoke the presented token and issue a fresh pair, so a stolen
	// refresh token can only be replayed once before detection.
	if err := s.store.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, rt.UserID)
}

func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := hashToken(rawRefreshToken)
	rt, err := s.store.GetRefreshToken(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	return s.store.RevokeRefreshToken(ctx, rt.ID)
}

func (s *Service) issueTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
	access, err := GenerateAccessToken(s.accessSecret, userID, s.accessTokenTTL)
	if err != nil {
		return nil, err
	}

	rawRefresh, hash, err := generateOpaqueToken()
	if err != nil {
		return nil, err
	}
	if err := s.store.CreateRefreshToken(ctx, uuid.NewString(), userID, hash, time.Now().Add(s.refreshTokenTTL)); err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: access, RefreshToken: rawRefresh}, nil
}

func (s *Service) ParseAccessToken(tokenStr string) (*AccessClaims, error) {
	return ParseAccessToken(s.accessSecret, tokenStr)
}

var ErrInvalidResetCode = errors.New("invalid or expired reset code")

const resetCodeTTL = 15 * time.Minute

// RequestPasswordReset emails a 6-digit code to the account, if it exists.
// It intentionally succeeds silently for unknown emails so the endpoint
// can't be used to probe which addresses have accounts.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}

	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if err := s.store.CreatePasswordResetCode(ctx, uuid.NewString(), user.ID, hashToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	return s.mailer.SendPasswordResetCode(user.Email, code)
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}

	rc, err := s.store.GetPasswordResetCode(ctx, user.ID, hashToken(code))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if rc.UsedAt != nil || time.Now().After(rc.ExpiresAt) {
		return ErrInvalidResetCode
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.store.MarkPasswordResetCodeUsed(ctx, rc.ID); err != nil {
		return err
	}
	if err := s.store.UpdatePassword(ctx, user.ID, hash); err != nil {
		return err
	}
	// Force every existing session to re-authenticate with the new password.
	return s.store.RevokeAllRefreshTokens(ctx, user.ID)
}
