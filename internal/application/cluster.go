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

type ClusterService struct {
	reader ClusterReader
}

func NewClusterService(reader ClusterReader) (*ClusterService, error) {
	if reader == nil {
		return nil, errors.New("cluster reader is required")
	}

	return &ClusterService{reader: reader}, nil
}

func (s *ClusterService) Snapshot(ctx context.Context) (domain.ClusterSnapshot, error) {
	return s.reader.Inspect(ctx)
}
