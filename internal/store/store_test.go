// SPDX-License-Identifier: AGPL-3.0-only

package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/nectarops/nectar/internal/domain"
)

func TestSetupAndSessionLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "nectar.db")
	storage, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	completed, err := storage.SetupCompleted(ctx)
	if err != nil {
		t.Fatalf("SetupCompleted() error = %v", err)
	}
	if completed {
		t.Fatal("SetupCompleted() = true before setup")
	}

	user, err := storage.CompleteSetup(
		ctx,
		"admin",
		"encoded-password",
		domain.ManagementAccess{
			Domain: "nectar.example.com", ACMEEmail: "ops@example.com",
		},
	)
	if err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	if user.Role != "owner" {
		t.Fatalf("CompleteSetup() role = %q, want owner", user.Role)
	}
	var managementDomain string
	if err := storage.db.QueryRowContext(
		ctx,
		"SELECT value FROM settings WHERE key = 'management_domain'",
	).Scan(&managementDomain); err != nil {
		t.Fatalf("read management domain setting: %v", err)
	}
	if managementDomain != "nectar.example.com" {
		t.Fatalf("management domain = %q, want nectar.example.com", managementDomain)
	}
	if _, err := storage.CompleteSetup(ctx, "second", "encoded-password", domain.ManagementAccess{}); !errors.Is(err, domain.ErrAlreadyConfigured) {
		t.Fatalf("second CompleteSetup() error = %v, want ErrAlreadyConfigured", err)
	}

	tokenHash := []byte("01234567890123456789012345678901")
	if err := storage.CreateSession(ctx, user.ID, tokenHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	authenticated, err := storage.UserBySession(ctx, tokenHash)
	if err != nil {
		t.Fatalf("UserBySession() error = %v", err)
	}
	if authenticated.ID != user.ID {
		t.Fatalf("UserBySession() ID = %d, want %d", authenticated.ID, user.ID)
	}
	if err := storage.DeleteSession(ctx, tokenHash); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	if _, err := storage.UserBySession(ctx, tokenHash); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("deleted UserBySession() error = %v, want ErrUnauthenticated", err)
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "nectar.db")
	first, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}

	second, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("second Open() error = %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestDesiredDockerVersionPolicyCannotBeSilentlyOverwritten(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	storage, err := Open(ctx, filepath.Join(t.TempDir(), "nectar.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	version, err := storage.DesiredDockerVersion(ctx)
	if err != nil {
		t.Fatalf("DesiredDockerVersion() before initialization error = %v", err)
	}
	if version != "" {
		t.Fatalf("DesiredDockerVersion() before initialization = %q, want empty", version)
	}

	if err := storage.EnsureDesiredDockerVersion(ctx, "28.3.0"); err != nil {
		t.Fatalf("EnsureDesiredDockerVersion() error = %v", err)
	}
	if err := storage.EnsureDesiredDockerVersion(ctx, "28.3.0"); err != nil {
		t.Fatalf("idempotent EnsureDesiredDockerVersion() error = %v", err)
	}
	if err := storage.EnsureDesiredDockerVersion(ctx, "29.0.2"); !errors.Is(err, domain.ErrDockerVersionConflict) {
		t.Fatalf("conflicting EnsureDesiredDockerVersion() error = %v, want ErrDockerVersionConflict", err)
	}

	version, err = storage.DesiredDockerVersion(ctx)
	if err != nil {
		t.Fatalf("DesiredDockerVersion() error = %v", err)
	}
	if version != "28.3.0" {
		t.Fatalf("DesiredDockerVersion() = %q, want 28.3.0", version)
	}
}
