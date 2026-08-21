// SPDX-License-Identifier: AGPL-3.0-only

package docker

import (
	"context"
	"errors"

	"github.com/ranen/dock-weaver/internal/domain"
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
