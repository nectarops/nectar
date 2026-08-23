// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"errors"
	"testing"

	"github.com/nectarops/nectar/internal/domain"
)

type managementAccessStore struct {
	access domain.ManagementAccess
	saves  int
	events *[]string
}

func (s *managementAccessStore) ManagementAccess(context.Context) (domain.ManagementAccess, error) {
	return s.access, nil
}

func (s *managementAccessStore) SaveManagementAccess(_ context.Context, access domain.ManagementAccess) error {
	s.saves++
	s.access = access
	if s.events != nil {
		*s.events = append(*s.events, "save")
	}
	return nil
}

type managementAccessConfigurator struct {
	access domain.ManagementAccess
	calls  int
	err    error
	events *[]string
}

func (c *managementAccessConfigurator) ConfigureManagementAccess(
	_ context.Context,
	access domain.ManagementAccess,
) error {
	c.calls++
	c.access = access
	if c.events != nil {
		*c.events = append(*c.events, "configure")
	}
	return c.err
}

func TestConfigureManagementAccessNormalizesBeforePersisting(t *testing.T) {
	t.Parallel()

	events := []string{}
	storage := &managementAccessStore{events: &events}
	configurator := &managementAccessConfigurator{events: &events}
	service, err := NewManagementAccessService(storage, configurator)
	if err != nil {
		t.Fatalf("NewManagementAccessService() error = %v", err)
	}

	access, err := service.Configure(t.Context(), domain.ManagementAccess{
		Domain:    " Nectar.Example.com ",
		ACMEEmail: " OPS@Example.com ",
	})
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	want := domain.ManagementAccess{Domain: "nectar.example.com", ACMEEmail: "ops@example.com"}
	if access != want || configurator.access != want || storage.access != want {
		t.Fatalf("management access = %#v/%#v/%#v, want %#v", access, configurator.access, storage.access, want)
	}
	if len(events) != 2 || events[0] != "configure" || events[1] != "save" {
		t.Fatalf("events = %#v, want configure then save", events)
	}
}

func TestConfigureManagementAccessRejectsPartialInput(t *testing.T) {
	t.Parallel()

	storage := &managementAccessStore{}
	configurator := &managementAccessConfigurator{}
	service, err := NewManagementAccessService(storage, configurator)
	if err != nil {
		t.Fatalf("NewManagementAccessService() error = %v", err)
	}

	_, err = service.Configure(t.Context(), domain.ManagementAccess{Domain: "nectar.example.com"})
	if !errors.Is(err, domain.ErrInvalidManagementAccess) {
		t.Fatalf("Configure() error = %v, want ErrInvalidManagementAccess", err)
	}
	if configurator.calls != 0 || storage.saves != 0 {
		t.Fatalf("invalid access configured=%d saved=%d", configurator.calls, storage.saves)
	}
}

func TestConfigureManagementAccessDoesNotPersistAfterTraefikFailure(t *testing.T) {
	t.Parallel()

	storage := &managementAccessStore{}
	configurator := &managementAccessConfigurator{err: errors.New("Traefik unavailable")}
	service, err := NewManagementAccessService(storage, configurator)
	if err != nil {
		t.Fatalf("NewManagementAccessService() error = %v", err)
	}

	_, err = service.Configure(t.Context(), domain.ManagementAccess{
		Domain: "nectar.example.com", ACMEEmail: "ops@example.com",
	})
	if err == nil {
		t.Fatal("Configure() succeeded after Traefik failure")
	}
	if storage.saves != 0 {
		t.Fatalf("SaveManagementAccess() calls = %d, want 0", storage.saves)
	}
}
