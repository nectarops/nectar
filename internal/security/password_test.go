// SPDX-License-Identifier: AGPL-3.0-only

package security

import (
	"strings"
	"testing"
)

func TestPasswordRoundTrip(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("a strong password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !VerifyPassword(hash, "a strong password") {
		t.Fatal("VerifyPassword() rejected the original password")
	}
	if VerifyPassword(hash, "a different password") {
		t.Fatal("VerifyPassword() accepted a different password")
	}
}

func TestHashPasswordRejectsShortPassword(t *testing.T) {
	t.Parallel()

	if _, err := HashPassword("too short"); err == nil {
		t.Fatal("HashPassword() accepted a short password")
	}
}

func TestVerifyPasswordRejectsUnsafeParameters(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("a strong password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	unsafeHash := strings.Replace(hash, "m=65536", "m=2097152", 1)
	if VerifyPassword(unsafeHash, "a strong password") {
		t.Fatal("VerifyPassword() accepted an unsafe memory cost")
	}
}
