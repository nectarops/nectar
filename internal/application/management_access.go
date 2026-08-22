// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/nectarops/nectar/internal/domain"
)

func normalizeManagementAccess(input domain.ManagementAccess) (domain.ManagementAccess, error) {
	access := domain.ManagementAccess{
		Domain:    strings.TrimSpace(strings.ToLower(input.Domain)),
		ACMEEmail: strings.TrimSpace(strings.ToLower(input.ACMEEmail)),
	}
	if access.Domain == "" && access.ACMEEmail == "" {
		return domain.ManagementAccess{}, nil
	}
	if access.Domain == "" || access.ACMEEmail == "" {
		return domain.ManagementAccess{}, errors.New(
			"management domain and Let's Encrypt email must be provided together",
		)
	}
	if !domainPattern.MatchString(access.Domain) {
		return domain.ManagementAccess{}, errors.New("enter a valid management domain")
	}

	address, err := mail.ParseAddress(access.ACMEEmail)
	if err != nil || address.Address != access.ACMEEmail {
		return domain.ManagementAccess{}, errors.New("enter a valid Let's Encrypt email address")
	}

	return access, nil
}
