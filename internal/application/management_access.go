// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/nectarops/nectar/internal/domain"
)

type ManagementAccessStore interface {
	ManagementAccess(context.Context) (domain.ManagementAccess, error)
	SaveManagementAccess(context.Context, domain.ManagementAccess) error
}

type ManagementAccessConfigurator interface {
	ConfigureManagementAccess(context.Context, domain.ManagementAccess) error
}

type ManagementAccessService struct {
	store        ManagementAccessStore
	configurator ManagementAccessConfigurator
}

func NewManagementAccessService(
	store ManagementAccessStore,
	configurator ManagementAccessConfigurator,
) (*ManagementAccessService, error) {
	if store == nil {
		return nil, errors.New("management access store is required")
	}
	if configurator == nil {
		return nil, errors.New("management access configurator is required")
	}
	return &ManagementAccessService{store: store, configurator: configurator}, nil
}

func (s *ManagementAccessService) Current(ctx context.Context) (domain.ManagementAccess, error) {
	return s.store.ManagementAccess(ctx)
}

func (s *ManagementAccessService) Configure(
	ctx context.Context,
	input domain.ManagementAccess,
) (domain.ManagementAccess, error) {
	access, err := normalizeManagementAccess(input)
	if err != nil {
		return domain.ManagementAccess{}, err
	}
	if access.Domain == "" {
		return domain.ManagementAccess{}, fmt.Errorf(
			"%w: management domain and Let's Encrypt email are required",
			domain.ErrInvalidManagementAccess,
		)
	}
	if err := s.configurator.ConfigureManagementAccess(ctx, access); err != nil {
		return domain.ManagementAccess{}, fmt.Errorf("configure Traefik management access: %w", err)
	}
	if err := s.store.SaveManagementAccess(ctx, access); err != nil {
		return domain.ManagementAccess{}, fmt.Errorf("persist management access: %w", err)
	}
	return access, nil
}

func normalizeManagementAccess(input domain.ManagementAccess) (domain.ManagementAccess, error) {
	access := domain.ManagementAccess{
		Domain:    strings.TrimSpace(strings.ToLower(input.Domain)),
		ACMEEmail: strings.TrimSpace(strings.ToLower(input.ACMEEmail)),
	}
	if access.Domain == "" && access.ACMEEmail == "" {
		return domain.ManagementAccess{}, nil
	}
	if access.Domain == "" || access.ACMEEmail == "" {
		return domain.ManagementAccess{}, fmt.Errorf(
			"%w: management domain and Let's Encrypt email must be provided together",
			domain.ErrInvalidManagementAccess,
		)
	}
	if !domainPattern.MatchString(access.Domain) {
		return domain.ManagementAccess{}, fmt.Errorf(
			"%w: enter a valid management domain",
			domain.ErrInvalidManagementAccess,
		)
	}

	address, err := mail.ParseAddress(access.ACMEEmail)
	if err != nil || address.Address != access.ACMEEmail {
		return domain.ManagementAccess{}, fmt.Errorf(
			"%w: enter a valid Let's Encrypt email address",
			domain.ErrInvalidManagementAccess,
		)
	}

	return access, nil
}
