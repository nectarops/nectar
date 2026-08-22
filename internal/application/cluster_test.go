// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"errors"
	"testing"

	"github.com/nectarops/nectar/internal/domain"
)

func TestClusterSnapshotIncludesDesiredDockerVersion(t *testing.T) {
	t.Parallel()

	service, err := NewClusterService(
		clusterReaderStub{snapshot: domain.ClusterSnapshot{DockerVersion: "28.3.0"}},
		dockerVersionPolicyReaderStub{version: "28.3.0"},
	)
	if err != nil {
		t.Fatalf("NewClusterService() error = %v", err)
	}

	snapshot, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if snapshot.DesiredDockerVersion != "28.3.0" {
		t.Fatalf("Snapshot() desired Docker version = %q, want 28.3.0", snapshot.DesiredDockerVersion)
	}
}

func TestClusterSnapshotReturnsDockerVersionPolicyError(t *testing.T) {
	t.Parallel()

	policyErr := errors.New("read policy")
	service, err := NewClusterService(
		clusterReaderStub{},
		dockerVersionPolicyReaderStub{err: policyErr},
	)
	if err != nil {
		t.Fatalf("NewClusterService() error = %v", err)
	}

	if _, err := service.Snapshot(context.Background()); !errors.Is(err, policyErr) {
		t.Fatalf("Snapshot() error = %v, want %v", err, policyErr)
	}
}

type clusterReaderStub struct {
	snapshot domain.ClusterSnapshot
	err      error
}

func (s clusterReaderStub) Inspect(context.Context) (domain.ClusterSnapshot, error) {
	return s.snapshot, s.err
}

type dockerVersionPolicyReaderStub struct {
	version string
	err     error
}

func (s dockerVersionPolicyReaderStub) DesiredDockerVersion(context.Context) (string, error) {
	return s.version, s.err
}
