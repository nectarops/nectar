// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"testing"

	"github.com/nectarops/nectar/internal/domain"
)

type recordingDeploymentEngine struct {
	called bool
	spec   domain.DeploymentSpec
}

func (e *recordingDeploymentEngine) Deploy(
	_ context.Context,
	spec domain.DeploymentSpec,
) (domain.DeploymentResult, error) {
	e.called = true
	e.spec = spec
	return domain.DeploymentResult{ServiceID: "service-id", Image: spec.Image + ":" + spec.Version}, nil
}

func TestDeploymentValidationAndNormalization(t *testing.T) {
	t.Parallel()

	engine := &recordingDeploymentEngine{}
	service, err := NewDeploymentService(engine)
	if err != nil {
		t.Fatalf("NewDeploymentService() error = %v", err)
	}

	result, err := service.Deploy(t.Context(), domain.DeploymentSpec{
		ServiceName: " Payments-API ",
		Image:       "ghcr.io/acme/payments",
		Version:     "1.4.2",
		Domain:      "API.Example.com",
		ACMEEmail:   "OPS@Example.com",
		Port:        8080,
		Replicas:    2,
	})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if !engine.called {
		t.Fatal("deployment engine was not called")
	}
	if engine.spec.ServiceName != "payments-api" || engine.spec.Domain != "api.example.com" {
		t.Fatalf("normalized spec = %#v", engine.spec)
	}
	if result.Warnings == nil {
		t.Fatal("Deploy() returned nil warnings")
	}
}

func TestDeploymentRejectsLatest(t *testing.T) {
	t.Parallel()

	engine := &recordingDeploymentEngine{}
	service, err := NewDeploymentService(engine)
	if err != nil {
		t.Fatalf("NewDeploymentService() error = %v", err)
	}

	_, err = service.Deploy(t.Context(), domain.DeploymentSpec{
		ServiceName: "payments-api",
		Image:       "ghcr.io/acme/payments",
		Version:     "latest",
		Domain:      "api.example.com",
		ACMEEmail:   "ops@example.com",
		Port:        8080,
		Replicas:    1,
	})
	if err == nil {
		t.Fatal("Deploy() accepted the latest tag")
	}
	if engine.called {
		t.Fatal("deployment engine was called for invalid input")
	}
}

func TestDeploymentRejectsLabelInjection(t *testing.T) {
	t.Parallel()

	engine := &recordingDeploymentEngine{}
	service, err := NewDeploymentService(engine)
	if err != nil {
		t.Fatalf("NewDeploymentService() error = %v", err)
	}

	_, err = service.Deploy(t.Context(), domain.DeploymentSpec{
		ServiceName: "payments-api",
		Image:       "ghcr.io/acme/payments",
		Version:     "1.4.2",
		Domain:      "example.com`) || Host(`attacker.example",
		ACMEEmail:   "ops@example.com",
		Port:        8080,
		Replicas:    1,
	})
	if err == nil {
		t.Fatal("Deploy() accepted a Traefik rule injection")
	}
}
