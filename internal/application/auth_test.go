// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nectarops/nectar/internal/domain"
)

type setupStore struct {
	completed bool
}

func (s *setupStore) SetupCompleted(context.Context) (bool, error) {
	return s.completed, nil
}

func (s *setupStore) CompleteSetup(
	_ context.Context,
	_ string,
	_ string,
) (domain.User, error) {
	s.completed = true
	return domain.User{ID: 1, Username: "admin", Role: "owner"}, nil
}

func (s *setupStore) UserByUsername(context.Context, string) (domain.StoredUser, error) {
	return domain.StoredUser{}, errors.New("not implemented")
}

func (s *setupStore) CreateSession(context.Context, int64, []byte, time.Time) error {
	return nil
}

func (s *setupStore) UserBySession(context.Context, []byte) (domain.User, error) {
	return domain.User{}, errors.New("not implemented")
}

func (s *setupStore) DeleteSession(context.Context, []byte) error {
	return errors.New("not implemented")
}

func TestSetupCreatesOwnerBeforeHTTPSConfiguration(t *testing.T) {
	t.Parallel()

	storage := &setupStore{}
	service, err := NewAuthService(storage, "setup-token", time.Hour)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	result, err := service.Setup(t.Context(), SetupInput{
		InitToken: "setup-token",
		Username:  "admin",
		Password:  "abcde",
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if !storage.completed {
		t.Fatal("Setup() did not persist the owner")
	}
	if result.User.Role != "owner" {
		t.Fatalf("Setup() role = %q, want owner", result.User.Role)
	}
}
