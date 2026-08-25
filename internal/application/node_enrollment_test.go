// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nectarops/nectar/internal/domain"
	"github.com/nectarops/nectar/internal/store"
)

func TestWorkerEnrollmentPreservesExistingDockerVersion(t *testing.T) {
	t.Parallel()

	service, cluster, ownerID := newNodeEnrollmentTestService(t)
	credential, err := service.Create(context.Background(), ownerID, domain.NodeRoleWorker)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	claim := validNodeClaim("27.5.1")
	bootstrap, err := service.Claim(context.Background(), credential.Token, claim)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if bootstrap.DockerTargetVersion != "28.3.0" {
		t.Fatalf("Claim() Docker target = %q, want 28.3.0", bootstrap.DockerTargetVersion)
	}
	if bootstrap.WorkerJoinToken != "SWMTKN-worker-token" {
		t.Fatal("Claim() did not return the Worker join credential")
	}
	advanceEnrollmentToVerifying(t, service, credential.Token, claim)

	cluster.node = readyNode(domain.NodeRoleWorker, "27.5.1")
	completed, err := service.Complete(
		context.Background(),
		credential.Token,
		claim.MachineIDHash,
		cluster.node.ID,
	)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if completed.Status != domain.NodeEnrollmentCompleted {
		t.Fatalf("Complete() status = %q, want completed", completed.Status)
	}
	if completed.DockerVersion != "27.5.1" {
		t.Fatalf("Complete() Docker version = %q, want preserved 27.5.1", completed.DockerVersion)
	}
	if cluster.promotions != 0 {
		t.Fatalf("Complete() promotions = %d, want 0", cluster.promotions)
	}
	repeated, err := service.Complete(
		context.Background(),
		credential.Token,
		claim.MachineIDHash,
		cluster.node.ID,
	)
	if err != nil || repeated.Status != domain.NodeEnrollmentCompleted {
		t.Fatalf("repeated Complete() = %#v, %v; want completed", repeated, err)
	}
}

func TestManagerEnrollmentBlocksPromotionOnDockerVersionDrift(t *testing.T) {
	t.Parallel()

	service, cluster, ownerID := newNodeEnrollmentTestService(t)
	credential, err := service.Create(context.Background(), ownerID, domain.NodeRoleManager)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	claim := validNodeClaim("27.5.1")
	if _, err := service.Claim(context.Background(), credential.Token, claim); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	advanceEnrollmentToVerifying(t, service, credential.Token, claim)
	cluster.node = readyNode(domain.NodeRoleWorker, "27.5.1")

	completed, err := service.Complete(
		context.Background(),
		credential.Token,
		claim.MachineIDHash,
		cluster.node.ID,
	)
	if !errors.Is(err, domain.ErrManagerVersionMismatch) {
		t.Fatalf("Complete() error = %v, want ErrManagerVersionMismatch", err)
	}
	if completed.Status != domain.NodeEnrollmentPromotionBlocked {
		t.Fatalf("Complete() status = %q, want promotion_blocked", completed.Status)
	}
	if cluster.promotions != 0 {
		t.Fatalf("Complete() promotions = %d, want 0", cluster.promotions)
	}
}

func TestManagerEnrollmentPromotesReadyVersionMatchedWorker(t *testing.T) {
	t.Parallel()

	service, cluster, ownerID := newNodeEnrollmentTestService(t)
	credential, err := service.Create(context.Background(), ownerID, domain.NodeRoleManager)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	claim := validNodeClaim("28.3.0")
	if _, err := service.Claim(context.Background(), credential.Token, claim); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	advanceEnrollmentToVerifying(t, service, credential.Token, claim)
	cluster.node = readyNode(domain.NodeRoleWorker, "28.3.0")

	completed, err := service.Complete(
		context.Background(),
		credential.Token,
		claim.MachineIDHash,
		cluster.node.ID,
	)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if completed.Status != domain.NodeEnrollmentCompleted {
		t.Fatalf("Complete() status = %q, want completed", completed.Status)
	}
	if cluster.promotions != 1 {
		t.Fatalf("Complete() promotions = %d, want 1", cluster.promotions)
	}
}

func TestFailedEnrollmentCanResumeOnlyFromClaimedMachine(t *testing.T) {
	t.Parallel()

	service, _, ownerID := newNodeEnrollmentTestService(t)
	credential, err := service.Create(context.Background(), ownerID, domain.NodeRoleWorker)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	claim := validNodeClaim("")
	if _, err := service.Claim(context.Background(), credential.Token, claim); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if err := service.Progress(
		context.Background(),
		credential.Token,
		claim.MachineIDHash,
		"failed",
		"",
	); err != nil {
		t.Fatalf("Progress(failed) error = %v", err)
	}

	if _, err := service.Claim(context.Background(), credential.Token, claim); err != nil {
		t.Fatalf("retry Claim() error = %v", err)
	}
	otherMachine := claim
	otherMachine.MachineIDHash = strings.Repeat("b", 64)
	if _, err := service.Claim(context.Background(), credential.Token, otherMachine); !errors.Is(err, domain.ErrEnrollmentClaimed) {
		t.Fatalf("other-machine Claim() error = %v, want ErrEnrollmentClaimed", err)
	}
}

func newNodeEnrollmentTestService(
	t *testing.T,
) (*NodeEnrollmentService, *nodeClusterStub, int64) {
	t.Helper()
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "nectar.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	owner, err := database.CompleteSetup(ctx, "owner", "password-hash")
	if err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	if err := database.EnsureDesiredDockerVersion(ctx, "28.3.0"); err != nil {
		t.Fatalf("EnsureDesiredDockerVersion() error = %v", err)
	}
	cluster := &nodeClusterStub{}
	service, err := NewNodeEnrollmentService(database, cluster, database)
	if err != nil {
		t.Fatalf("NewNodeEnrollmentService() error = %v", err)
	}
	return service, cluster, owner.ID
}

func validNodeClaim(dockerVersion string) domain.NodeEnrollmentClaim {
	return domain.NodeEnrollmentClaim{
		Hostname:         "worker-1",
		MachineIDHash:    strings.Repeat("a", 64),
		OperatingSystem:  "Ubuntu 24.04 LTS",
		Architecture:     "amd64",
		AdvertiseAddress: "10.0.0.12",
		DataPathAddress:  "10.0.0.12",
		DockerVersion:    dockerVersion,
	}
}

func advanceEnrollmentToVerifying(
	t *testing.T,
	service *NodeEnrollmentService,
	token string,
	claim domain.NodeEnrollmentClaim,
) {
	t.Helper()
	if err := service.Progress(
		context.Background(),
		token,
		claim.MachineIDHash,
		"docker-ready",
		claim.DockerVersion,
	); err != nil {
		t.Fatalf("Progress(docker-ready) error = %v", err)
	}
	if err := service.Progress(
		context.Background(),
		token,
		claim.MachineIDHash,
		"verifying",
		claim.DockerVersion,
	); err != nil {
		t.Fatalf("Progress(verifying) error = %v", err)
	}
}

func readyNode(role domain.NodeRole, dockerVersion string) domain.SwarmNode {
	return domain.SwarmNode{
		ID:            "abcdefghijklmnopqrstuvwxy",
		Hostname:      "worker-1",
		Role:          role,
		Status:        "ready",
		DockerVersion: dockerVersion,
	}
}

type nodeClusterStub struct {
	node       domain.SwarmNode
	promotions int
}

func (s *nodeClusterStub) ListNodes(context.Context) ([]domain.SwarmNode, error) {
	return []domain.SwarmNode{s.node}, nil
}

func (s *nodeClusterStub) WorkerJoinConfiguration(context.Context) (string, string, string, error) {
	return "10.0.0.7:2377", "abcdefghijklmnopqrstuvwxy", "SWMTKN-worker-token", nil
}

func (s *nodeClusterStub) Node(context.Context, string) (domain.SwarmNode, error) {
	return s.node, nil
}

func (s *nodeClusterStub) PromoteNode(context.Context, string) (domain.SwarmNode, error) {
	s.promotions++
	s.node.Role = domain.NodeRoleManager
	s.node.ManagerStatus = "reachable"
	return s.node, nil
}
