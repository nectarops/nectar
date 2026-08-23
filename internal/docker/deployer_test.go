// SPDX-License-Identifier: AGPL-3.0-only

package docker

import (
	"reflect"
	"testing"

	"github.com/moby/moby/api/types/swarm"
)

func TestNewTraefikServiceSpecConfiguresPinnedHTTPSIngress(t *testing.T) {
	t.Parallel()

	spec := newTraefikServiceSpec("network-id", "ops@example.com")
	if spec.Annotations.Name != traefikServiceName {
		t.Fatalf("service name = %q, want %q", spec.Annotations.Name, traefikServiceName)
	}
	if spec.Annotations.Labels["io.nectar.managed"] != "true" {
		t.Fatal("Traefik service is missing the Nectar management label")
	}
	if spec.Annotations.Labels["io.nectar.acme-email"] != "ops@example.com" {
		t.Fatalf("ACME email label = %q", spec.Annotations.Labels["io.nectar.acme-email"])
	}

	container := spec.TaskTemplate.ContainerSpec
	if container == nil {
		t.Fatal("Traefik container spec is nil")
	}
	if container.Image != traefikImage {
		t.Fatalf("Traefik image = %q, want %q", container.Image, traefikImage)
	}
	if !container.ReadOnly {
		t.Fatal("Traefik root filesystem is not read-only")
	}

	args := make(map[string]bool, len(container.Args))
	for _, arg := range container.Args {
		args[arg] = true
	}
	for _, required := range []string{
		"--providers.swarm.exposedbydefault=false",
		"--entrypoints.web.http.redirections.entrypoint.to=websecure",
		"--entrypoints.websecure.address=:443",
		"--certificatesresolvers.letsencrypt.acme.email=ops@example.com",
		"--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json",
		"--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web",
	} {
		if !args[required] {
			t.Errorf("Traefik args are missing %q", required)
		}
	}

	if spec.Mode.Replicated == nil || spec.Mode.Replicated.Replicas == nil || *spec.Mode.Replicated.Replicas != 1 {
		t.Fatalf("Traefik replicas = %#v, want 1", spec.Mode.Replicated)
	}
	if spec.TaskTemplate.RestartPolicy == nil || spec.TaskTemplate.RestartPolicy.Condition != swarm.RestartPolicyConditionAny {
		t.Fatalf("Traefik restart policy = %#v, want any", spec.TaskTemplate.RestartPolicy)
	}
	wantConstraints := []string{
		"node.role == manager",
		"node.labels.nectar.control == true",
	}
	if spec.TaskTemplate.Placement == nil ||
		!reflect.DeepEqual(spec.TaskTemplate.Placement.Constraints, wantConstraints) {
		t.Fatalf("Traefik constraints = %#v, want %#v", spec.TaskTemplate.Placement, wantConstraints)
	}

	if len(spec.TaskTemplate.Networks) != 1 || spec.TaskTemplate.Networks[0].Target != "network-id" {
		t.Fatalf("Traefik networks = %#v", spec.TaskTemplate.Networks)
	}
	if spec.EndpointSpec == nil || len(spec.EndpointSpec.Ports) != 2 {
		t.Fatalf("Traefik ports = %#v, want 80 and 443", spec.EndpointSpec)
	}
	if spec.EndpointSpec.Ports[0].PublishedPort != 80 || spec.EndpointSpec.Ports[1].PublishedPort != 443 {
		t.Fatalf("published ports = %#v, want 80 and 443", spec.EndpointSpec.Ports)
	}
	for _, port := range spec.EndpointSpec.Ports {
		if port.PublishMode != swarm.PortConfigPublishModeHost {
			t.Fatalf("Traefik port %d publish mode = %s, want host", port.PublishedPort, port.PublishMode)
		}
	}
}
