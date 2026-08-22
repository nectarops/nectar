// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"errors"

	"github.com/nectarops/nectar/internal/domain"
)

type ClusterReader interface {
	Inspect(context.Context) (domain.ClusterSnapshot, error)
}

type DockerVersionPolicyReader interface {
	DesiredDockerVersion(context.Context) (string, error)
}

type ClusterService struct {
	reader       ClusterReader
	policyReader DockerVersionPolicyReader
}

func NewClusterService(
	reader ClusterReader,
	policyReader DockerVersionPolicyReader,
) (*ClusterService, error) {
	if reader == nil {
		return nil, errors.New("cluster reader is required")
	}
	if policyReader == nil {
		return nil, errors.New("Docker version policy reader is required")
	}

	return &ClusterService{reader: reader, policyReader: policyReader}, nil
}

func (s *ClusterService) Snapshot(ctx context.Context) (domain.ClusterSnapshot, error) {
	snapshot, err := s.reader.Inspect(ctx)
	if err != nil {
		return domain.ClusterSnapshot{}, err
	}

	snapshot.DesiredDockerVersion, err = s.policyReader.DesiredDockerVersion(ctx)
	if err != nil {
		return domain.ClusterSnapshot{}, err
	}
	return snapshot, nil
}
