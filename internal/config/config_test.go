// SPDX-License-Identifier: AGPL-3.0-only

package config

import "testing"

func TestLoadDesiredDockerVersion(t *testing.T) {
	t.Setenv("NECTAR_DESIRED_DOCKER_VERSION", " 28.3.0 ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DesiredDockerVersion != "28.3.0" {
		t.Fatalf("DesiredDockerVersion = %q, want 28.3.0", cfg.DesiredDockerVersion)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestRejectsInvalidDesiredDockerVersion(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.DesiredDockerVersion = "latest"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid Docker version error")
	}
}
