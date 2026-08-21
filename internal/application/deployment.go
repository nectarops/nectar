// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/ranen/dock-weaver/internal/domain"
)

var (
	serviceNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
	imagePattern       = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/-]{0,254}$`)
	versionPattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)
	domainPattern      = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+)$`)
)

type DeploymentEngine interface {
	Deploy(context.Context, domain.DeploymentSpec) (domain.DeploymentResult, error)
}

type DeploymentService struct {
	engine DeploymentEngine
}

func NewDeploymentService(engine DeploymentEngine) (*DeploymentService, error) {
	if engine == nil {
		return nil, errors.New("deployment engine is required")
	}
	return &DeploymentService{engine: engine}, nil
}

func (s *DeploymentService) Deploy(
	ctx context.Context,
	spec domain.DeploymentSpec,
) (domain.DeploymentResult, error) {
	spec.ServiceName = strings.TrimSpace(strings.ToLower(spec.ServiceName))
	spec.Image = strings.TrimSpace(spec.Image)
	spec.Version = strings.TrimSpace(spec.Version)
	spec.Domain = strings.TrimSpace(strings.ToLower(spec.Domain))
	spec.ACMEEmail = strings.TrimSpace(strings.ToLower(spec.ACMEEmail))

	if !serviceNamePattern.MatchString(spec.ServiceName) {
		return domain.DeploymentResult{}, errors.New("service name must be 3–63 lowercase letters, numbers, or dashes")
	}
	if !imagePattern.MatchString(spec.Image) || strings.Contains(spec.Image, "@") {
		return domain.DeploymentResult{}, errors.New("enter an image repository without a tag or digest")
	}
	if !versionPattern.MatchString(spec.Version) || strings.EqualFold(spec.Version, "latest") {
		return domain.DeploymentResult{}, errors.New("enter an explicit image version other than latest")
	}
	if !domainPattern.MatchString(spec.Domain) {
		return domain.DeploymentResult{}, errors.New("enter a valid fully qualified domain name")
	}
	address, err := mail.ParseAddress(spec.ACMEEmail)
	if err != nil || address.Address != spec.ACMEEmail {
		return domain.DeploymentResult{}, errors.New("enter a valid ACME email address")
	}
	if spec.Port == 0 || spec.Port > 65535 {
		return domain.DeploymentResult{}, errors.New("container port must be between 1 and 65535")
	}
	if spec.Replicas == 0 || spec.Replicas > 1000 {
		return domain.DeploymentResult{}, errors.New("replicas must be between 1 and 1000")
	}

	result, err := s.engine.Deploy(ctx, spec)
	if err != nil {
		return domain.DeploymentResult{}, fmt.Errorf("deploy service: %w", err)
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	return result, nil
}
