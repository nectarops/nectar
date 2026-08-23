// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nectarops/nectar/internal/domain"
	"github.com/nectarops/nectar/internal/security"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{2,31}$`)

type AuthStore interface {
	SetupCompleted(context.Context) (bool, error)
	CompleteSetup(context.Context, string, string) (domain.User, error)
	UserByUsername(context.Context, string) (domain.StoredUser, error)
	CreateSession(context.Context, int64, []byte, time.Time) error
	UserBySession(context.Context, []byte) (domain.User, error)
	DeleteSession(context.Context, []byte) error
}

type AuthService struct {
	store           AuthStore
	initToken       string
	sessionDuration time.Duration
}

type SetupInput struct {
	InitToken string
	Username  string
	Password  string
}

type AuthResult struct {
	User    domain.User
	Session domain.Session
}

func NewAuthService(
	store AuthStore,
	initToken string,
	sessionDuration time.Duration,
) (*AuthService, error) {
	if store == nil {
		return nil, errors.New("authentication store is required")
	}
	if sessionDuration <= 0 {
		return nil, errors.New("session duration must be positive")
	}

	return &AuthService{
		store:           store,
		initToken:       initToken,
		sessionDuration: sessionDuration,
	}, nil
}

func (s *AuthService) SetupStatus(ctx context.Context) (bool, error) {
	return s.store.SetupCompleted(ctx)
}

func (s *AuthService) Setup(ctx context.Context, input SetupInput) (AuthResult, error) {
	completed, err := s.store.SetupCompleted(ctx)
	if err != nil {
		return AuthResult{}, err
	}
	if completed {
		return AuthResult{}, domain.ErrAlreadyConfigured
	}
	if s.initToken == "" {
		return AuthResult{}, domain.ErrInitTokenMissing
	}
	if !constantTimeEqual(s.initToken, strings.TrimSpace(input.InitToken)) {
		return AuthResult{}, domain.ErrInvalidInitToken
	}

	username := strings.TrimSpace(input.Username)
	if !usernamePattern.MatchString(username) {
		return AuthResult{}, errors.New(
			"username must contain 3 to 32 letters, numbers, dots, underscores, or hyphens",
		)
	}

	passwordHash, err := security.HashPassword(input.Password)
	if err != nil {
		return AuthResult{}, err
	}

	user, err := s.store.CompleteSetup(ctx, username, passwordHash)
	if err != nil {
		return AuthResult{}, err
	}

	return s.createSession(ctx, user)
}

func (s *AuthService) Login(
	ctx context.Context,
	username string,
	password string,
) (AuthResult, error) {
	user, err := s.store.UserByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			security.VerifyPassword(
				"$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaA",
				password,
			)
		}
		return AuthResult{}, err
	}
	if !security.VerifyPassword(user.PasswordHash, password) {
		return AuthResult{}, domain.ErrInvalidCredentials
	}

	return s.createSession(ctx, user.User)
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (domain.User, error) {
	if token == "" {
		return domain.User{}, domain.ErrUnauthenticated
	}

	hash := security.HashToken(token)
	return s.store.UserBySession(ctx, hash[:])
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	hash := security.HashToken(token)
	return s.store.DeleteSession(ctx, hash[:])
}

func (s *AuthService) createSession(
	ctx context.Context,
	user domain.User,
) (AuthResult, error) {
	token, err := security.NewToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate session token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(s.sessionDuration)
	hash := security.HashToken(token)
	if err := s.store.CreateSession(ctx, user.ID, hash[:], expiresAt); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User: user,
		Session: domain.Session{
			Token:     token,
			ExpiresAt: expiresAt,
		},
	}, nil
}

func constantTimeEqual(expected, actual string) bool {
	expectedHash := security.HashToken(expected)
	actualHash := security.HashToken(actual)

	return subtle.ConstantTimeCompare(expectedHash[:], actualHash[:]) == 1
}
