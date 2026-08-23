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
	access    domain.ManagementAccess
}

func (s *setupStore) SetupCompleted(context.Context) (bool, error) {
	return s.completed, nil
}

func (s *setupStore) CompleteSetup(
	_ context.Context,
	_ string,
	_ string,
	access domain.ManagementAccess,
) (domain.User, error) {
	s.completed = true
	s.access = access
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

type recordingAccessConfigurator struct {
	access domain.ManagementAccess
	err    error
	calls  int
}

func (c *recordingAccessConfigurator) ConfigureManagementAccess(
	_ context.Context,
	access domain.ManagementAccess,
) error {
	c.calls++
	c.access = access
	return c.err
}

func TestSetupConfiguresNormalizedManagementAccess(t *testing.T) {
	t.Parallel()

	storage := &setupStore{}
	configurator := &recordingAccessConfigurator{}
	service, err := NewAuthService(storage, configurator, "setup-token", time.Hour)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	_, err = service.Setup(t.Context(), SetupInput{
		InitToken: "setup-token",
		Username:  "admin",
		Password:  "abcde",
		ManagementAccess: domain.ManagementAccess{
			Domain:    " Nectar.Example.com ",
			ACMEEmail: " OPS@Example.com ",
		},
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	want := domain.ManagementAccess{
		Domain:    "nectar.example.com",
		ACMEEmail: "ops@example.com",
	}
	if configurator.access != want {
		t.Fatalf("configured access = %#v, want %#v", configurator.access, want)
	}
	if configurator.calls != 1 {
		t.Fatalf("ConfigureManagementAccess() calls = %d, want 1", configurator.calls)
	}
	if storage.access != want {
		t.Fatalf("stored access = %#v, want %#v", storage.access, want)
	}
}

func TestSetupWithoutManagementAccessSkipsIngressConfiguration(t *testing.T) {
	t.Parallel()

	storage := &setupStore{}
	configurator := &recordingAccessConfigurator{}
	service, err := NewAuthService(storage, configurator, "setup-token", time.Hour)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	if _, err := service.Setup(t.Context(), SetupInput{
		InitToken: "setup-token",
		Username:  "admin",
		Password:  "abcde",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if configurator.calls != 0 {
		t.Fatalf("ConfigureManagementAccess() calls = %d, want 0", configurator.calls)
	}
	if !storage.completed {
		t.Fatal("Setup() did not persist the owner")
	}
}

func TestSetupRejectsPartialManagementAccess(t *testing.T) {
	t.Parallel()

	storage := &setupStore{}
	service, err := NewAuthService(storage, &recordingAccessConfigurator{}, "setup-token", time.Hour)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	_, err = service.Setup(t.Context(), SetupInput{
		InitToken: "setup-token",
		Username:  "admin",
		Password:  "abcde",
		ManagementAccess: domain.ManagementAccess{
			Domain: "nectar.example.com",
		},
	})
	if err == nil {
		t.Fatal("Setup() accepted a domain without an ACME email")
	}
	if storage.completed {
		t.Fatal("Setup() persisted the owner after management access validation failed")
	}
}

func TestSetupDoesNotPersistAfterIngressFailure(t *testing.T) {
	t.Parallel()

	storage := &setupStore{}
	configurator := &recordingAccessConfigurator{err: errors.New("Traefik unavailable")}
	service, err := NewAuthService(storage, configurator, "setup-token", time.Hour)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	_, err = service.Setup(t.Context(), SetupInput{
		InitToken: "setup-token",
		Username:  "admin",
		Password:  "abcde",
		ManagementAccess: domain.ManagementAccess{
			Domain:    "nectar.example.com",
			ACMEEmail: "ops@example.com",
		},
	})
	if err == nil {
		t.Fatal("Setup() succeeded after ingress configuration failed")
	}
	if storage.completed {
		t.Fatal("Setup() persisted the owner after ingress configuration failed")
	}
}
