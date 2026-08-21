// SPDX-License-Identifier: AGPL-3.0-only

package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ranen/dock-weaver/internal/domain"
)

func TestSetupAndSessionLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "dock-weaver.db")
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

	user, err := storage.CompleteSetup(ctx, "admin", "encoded-password")
	if err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	if user.Role != "owner" {
		t.Fatalf("CompleteSetup() role = %q, want owner", user.Role)
	}
	if _, err := storage.CompleteSetup(ctx, "second", "encoded-password"); !errors.Is(err, domain.ErrAlreadyConfigured) {
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
	databasePath := filepath.Join(t.TempDir(), "dock-weaver.db")
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
