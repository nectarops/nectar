// SPDX-License-Identifier: AGPL-3.0-only

package store

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nectarops/nectar/internal/domain"
)

func TestNodeEnrollmentPersistsHashedCredentialAndMachineBinding(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "nectar.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
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
	now := time.Now().UTC().Truncate(time.Second)
	enrollment := domain.NodeEnrollment{
		ID:            "ne_abcdefghijklmnopqrstuvwx",
		RequestedRole: domain.NodeRoleWorker,
		Status:        domain.NodeEnrollmentPending,
		Message:       "Enrollment command created",
		ExpiresAt:     now.Add(30 * time.Minute),
		CreatedBy:     owner.ID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	tokenHash := []byte("01234567890123456789012345678901")
	if err := database.CreateNodeEnrollment(ctx, enrollment, tokenHash); err != nil {
		t.Fatalf("CreateNodeEnrollment() error = %v", err)
	}
	stored, err := database.NodeEnrollmentByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("NodeEnrollmentByTokenHash() error = %v", err)
	}
	if stored.Message != enrollment.Message {
		t.Fatalf("stored message = %q, want %q", stored.Message, enrollment.Message)
	}
	if _, err := database.NodeEnrollmentByTokenHash(ctx, []byte("different-token-hash-value-12345")); !errors.Is(
		err,
		domain.ErrInvalidEnrollmentToken,
	) {
		t.Fatalf("unknown token error = %v, want ErrInvalidEnrollmentToken", err)
	}

	claim := domain.NodeEnrollmentClaim{
		Hostname:         "worker-1",
		MachineIDHash:    strings.Repeat("a", 64),
		OperatingSystem:  "Ubuntu 24.04 LTS",
		Architecture:     "amd64",
		AdvertiseAddress: "10.0.0.12",
		DataPathAddress:  "10.0.0.12",
		DockerVersion:    "28.3.0",
	}
	claimed, err := database.ClaimNodeEnrollment(ctx, enrollment.ID, claim, now.Add(time.Second))
	if err != nil {
		t.Fatalf("ClaimNodeEnrollment() error = %v", err)
	}
	if claimed.MachineIDHash != claim.MachineIDHash {
		t.Fatal("ClaimNodeEnrollment() did not bind the machine identity")
	}
	otherMachine := claim
	otherMachine.MachineIDHash = strings.Repeat("b", 64)
	if _, err := database.ClaimNodeEnrollment(
		ctx,
		enrollment.ID,
		otherMachine,
		now.Add(2*time.Second),
	); !errors.Is(err, domain.ErrEnrollmentClaimed) {
		t.Fatalf("other-machine claim error = %v, want ErrEnrollmentClaimed", err)
	}

	events, err := database.NodeEnrollmentEvents(ctx, enrollment.ID, 0)
	if err != nil {
		t.Fatalf("NodeEnrollmentEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("NodeEnrollmentEvents() count = %d, want 2", len(events))
	}
}
