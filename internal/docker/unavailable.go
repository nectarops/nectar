// SPDX-License-Identifier: AGPL-3.0-only

package docker

import (
	"context"
	"errors"

	"github.com/nectarops/nectar/internal/domain"
)

type UnavailableInspector struct {
	reason string
}

func NewUnavailableInspector(err error) *UnavailableInspector {
	reason := "Docker Engine is unavailable"
	if err != nil {
		reason = err.Error()
	}

	return &UnavailableInspector{reason: reason}
}

func (i *UnavailableInspector) Inspect(context.Context) (domain.ClusterSnapshot, error) {
	return domain.ClusterSnapshot{
		Available:  false,
		Error:      i.reason,
		SwarmState: "unavailable",
		NodeRole:   "unknown",
		NodeStatus: "unknown",
	}, nil
}

func (i *UnavailableInspector) Deploy(
	context.Context,
	domain.DeploymentSpec,
) (domain.DeploymentResult, error) {
	return domain.DeploymentResult{}, errors.New(i.reason)
}

func (i *UnavailableInspector) ConfigureManagementAccess(
	context.Context,
	domain.ManagementAccess,
) error {
	return errors.New(i.reason)
}

func (i *UnavailableInspector) ListNodes(context.Context) ([]domain.SwarmNode, error) {
	return nil, errors.New(i.reason)
}

func (i *UnavailableInspector) WorkerJoinConfiguration(context.Context) (string, string, string, error) {
	return "", "", "", errors.New(i.reason)
}

func (i *UnavailableInspector) Node(context.Context, string) (domain.SwarmNode, error) {
	return domain.SwarmNode{}, errors.New(i.reason)
}

func (i *UnavailableInspector) PromoteNode(context.Context, string) (domain.SwarmNode, error) {
	return domain.SwarmNode{}, errors.New(i.reason)
}
