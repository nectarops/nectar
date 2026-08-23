// SPDX-License-Identifier: AGPL-3.0-only

package docker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"

	"github.com/nectarops/nectar/internal/domain"
)

const (
	ingressNetworkName = "traefik-public"
	nectarServiceName  = "nectar_nectar"
	traefikServiceName = "nectar-traefik"
	traefikVolumeName  = "nectar-traefik-acme"
	traefikImage       = "traefik:v3.7.1"
)

func (i *Inspector) Deploy(
	ctx context.Context,
	spec domain.DeploymentSpec,
) (domain.DeploymentResult, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	networkID, err := i.ensureIngressNetwork(requestCtx)
	if err != nil {
		return domain.DeploymentResult{}, err
	}
	if err := i.ensureTraefik(requestCtx, networkID, spec.ACMEEmail); err != nil {
		return domain.DeploymentResult{}, err
	}

	image := spec.Image + ":" + spec.Version
	serviceSpec := applicationServiceSpec(spec, image, networkID)
	inspected, err := i.client.ServiceInspect(
		requestCtx,
		spec.ServiceName,
		client.ServiceInspectOptions{},
	)
	if err != nil && !cerrdefs.IsNotFound(err) {
		return domain.DeploymentResult{}, fmt.Errorf("inspect service: %w", err)
	}
	if cerrdefs.IsNotFound(err) {
		created, createErr := i.client.ServiceCreate(requestCtx, client.ServiceCreateOptions{
			Spec:          serviceSpec,
			QueryRegistry: true,
		})
		if createErr != nil {
			return domain.DeploymentResult{}, fmt.Errorf("create service: %w", createErr)
		}
		return domain.DeploymentResult{
			ServiceID: created.ID,
			Image:     image,
			Warnings:  created.Warnings,
		}, nil
	}
	if inspected.Service.Spec.Labels["io.nectar.managed"] != "true" {
		return domain.DeploymentResult{}, errors.New("a service with this name exists but is not managed by Nectar")
	}

	updated, err := i.client.ServiceUpdate(requestCtx, inspected.Service.ID, client.ServiceUpdateOptions{
		Version:       inspected.Service.Version,
		Spec:          serviceSpec,
		QueryRegistry: true,
	})
	if err != nil {
		return domain.DeploymentResult{}, fmt.Errorf("update service: %w", err)
	}
	return domain.DeploymentResult{
		ServiceID: inspected.Service.ID,
		Image:     image,
		Updated:   true,
		Warnings:  updated.Warnings,
	}, nil
}

func (i *Inspector) ConfigureManagementAccess(
	ctx context.Context,
	access domain.ManagementAccess,
) error {
	requestCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	networkID, err := i.ensureIngressNetwork(requestCtx)
	if err != nil {
		return err
	}
	if err := i.ensureTraefik(requestCtx, networkID, access.ACMEEmail); err != nil {
		return err
	}

	inspected, err := i.client.ServiceInspect(
		requestCtx,
		nectarServiceName,
		client.ServiceInspectOptions{},
	)
	if cerrdefs.IsNotFound(err) {
		return errors.New("installed Nectar Swarm service was not found")
	}
	if err != nil {
		return fmt.Errorf("inspect Nectar service: %w", err)
	}
	if inspected.Service.Spec.Labels["io.nectar.managed"] != "true" {
		return errors.New("installed Nectar service is not managed by Nectar")
	}

	spec := inspected.Service.Spec
	labels := make(map[string]string, len(spec.Labels)+8)
	for key, value := range spec.Labels {
		labels[key] = value
	}
	labels["io.nectar.management-domain"] = access.Domain
	labels["traefik.enable"] = "true"
	labels["traefik.swarm.network"] = ingressNetworkName
	labels["traefik.http.routers.nectar.rule"] = "Host(`" + access.Domain + "`)"
	labels["traefik.http.routers.nectar.entrypoints"] = "websecure"
	labels["traefik.http.routers.nectar.tls"] = "true"
	labels["traefik.http.routers.nectar.tls.certresolver"] = "letsencrypt"
	labels["traefik.http.services.nectar.loadbalancer.server.port"] = "8080"
	spec.Labels = labels

	attached := false
	for _, attachment := range spec.TaskTemplate.Networks {
		if attachment.Target == networkID || attachment.Target == ingressNetworkName {
			attached = true
			break
		}
	}
	if !attached {
		spec.TaskTemplate.Networks = append(
			spec.TaskTemplate.Networks,
			swarm.NetworkAttachmentConfig{Target: networkID},
		)
	}

	if _, err := i.client.ServiceUpdate(requestCtx, inspected.Service.ID, client.ServiceUpdateOptions{
		Version: inspected.Service.Version,
		Spec:    spec,
	}); err != nil {
		return fmt.Errorf("configure Nectar HTTPS route: %w", err)
	}
	return nil
}

func (i *Inspector) ensureIngressNetwork(ctx context.Context) (string, error) {
	inspected, err := i.client.NetworkInspect(
		ctx,
		ingressNetworkName,
		client.NetworkInspectOptions{Scope: "swarm"},
	)
	if err == nil {
		if inspected.Network.Driver != "overlay" || inspected.Network.Scope != "swarm" {
			return "", errors.New("traefik-public exists but is not a Swarm overlay network")
		}
		return inspected.Network.ID, nil
	}
	if !cerrdefs.IsNotFound(err) {
		return "", fmt.Errorf("inspect ingress network: %w", err)
	}

	created, err := i.client.NetworkCreate(ctx, ingressNetworkName, client.NetworkCreateOptions{
		Driver:     "overlay",
		Scope:      "swarm",
		Attachable: true,
		Labels:     map[string]string{"io.nectar.managed": "true"},
	})
	if err != nil {
		return "", fmt.Errorf("create ingress network: %w", err)
	}
	return created.ID, nil
}

func (i *Inspector) ensureTraefik(ctx context.Context, networkID, acmeEmail string) error {
	inspected, err := i.client.ServiceInspect(
		ctx,
		traefikServiceName,
		client.ServiceInspectOptions{},
	)
	if err == nil {
		if inspected.Service.Spec.Labels["io.nectar.managed"] != "true" {
			return errors.New("nectar-traefik exists but is not managed by Nectar")
		}
		configuredEmail := inspected.Service.Spec.Labels["io.nectar.acme-email"]
		if configuredEmail != "" && configuredEmail != acmeEmail {
			return fmt.Errorf(
				"Traefik is configured for ACME email %q; use the same email",
				configuredEmail,
			)
		}
		if _, err := i.client.ServiceUpdate(ctx, inspected.Service.ID, client.ServiceUpdateOptions{
			Version: inspected.Service.Version,
			Spec:    newTraefikServiceSpec(networkID, acmeEmail),
		}); err != nil {
			return fmt.Errorf("reconcile Traefik service: %w", err)
		}
		return nil
	}
	if !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("inspect Traefik service: %w", err)
	}

	if _, err := i.client.VolumeInspect(ctx, traefikVolumeName, client.VolumeInspectOptions{}); err != nil {
		if !cerrdefs.IsNotFound(err) {
			return fmt.Errorf("inspect Traefik ACME volume: %w", err)
		}
		if _, err := i.client.VolumeCreate(ctx, client.VolumeCreateOptions{
			Name:   traefikVolumeName,
			Labels: map[string]string{"io.nectar.managed": "true"},
		}); err != nil {
			return fmt.Errorf("create Traefik ACME volume: %w", err)
		}
	}

	if _, err := i.client.ServiceCreate(ctx, client.ServiceCreateOptions{
		Spec: newTraefikServiceSpec(networkID, acmeEmail),
	}); err != nil {
		return fmt.Errorf("create Traefik service: %w", err)
	}
	return nil
}

func newTraefikServiceSpec(networkID, acmeEmail string) swarm.ServiceSpec {
	replicas := uint64(1)
	delay := 5 * time.Second
	return swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: traefikServiceName,
			Labels: map[string]string{
				"io.nectar.managed":    "true",
				"io.nectar.acme-email": acmeEmail,
			},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: traefikImage,
				Args: []string{
					"--api.dashboard=false",
					"--log.level=INFO",
					"--providers.swarm.endpoint=unix:///var/run/docker.sock",
					"--providers.swarm.exposedbydefault=false",
					"--providers.swarm.network=" + ingressNetworkName,
					"--entrypoints.web.address=:80",
					"--entrypoints.web.http.redirections.entrypoint.to=websecure",
					"--entrypoints.web.http.redirections.entrypoint.scheme=https",
					"--entrypoints.websecure.address=:443",
					"--certificatesresolvers.letsencrypt.acme.email=" + acmeEmail,
					"--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json",
					"--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web",
				},
				Mounts: []mount.Mount{
					{
						Type: mount.TypeBind, Source: "/var/run/docker.sock",
						Target: "/var/run/docker.sock", ReadOnly: true,
					},
					{Type: mount.TypeVolume, Source: traefikVolumeName, Target: "/letsencrypt"},
				},
				ReadOnly: true,
			},
			RestartPolicy: &swarm.RestartPolicy{
				Condition: swarm.RestartPolicyConditionAny,
				Delay:     &delay,
			},
			Placement: &swarm.Placement{Constraints: []string{
				"node.role == manager",
				"node.labels.nectar.control == true",
			}},
			Networks: []swarm.NetworkAttachmentConfig{{Target: networkID}},
		},
		Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
		EndpointSpec: &swarm.EndpointSpec{Ports: []swarm.PortConfig{
			{
				Protocol: network.TCP, TargetPort: 80,
				PublishedPort: 80, PublishMode: swarm.PortConfigPublishModeHost,
			},
			{
				Protocol: network.TCP, TargetPort: 443,
				PublishedPort: 443, PublishMode: swarm.PortConfigPublishModeHost,
			},
		}},
	}
}

func applicationServiceSpec(spec domain.DeploymentSpec, image, networkID string) swarm.ServiceSpec {
	replicas := spec.Replicas
	delay := 5 * time.Second
	maxAttempts := uint64(3)
	router := spec.ServiceName
	return swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: spec.ServiceName,
			Labels: map[string]string{
				"io.nectar.managed":                                             "true",
				"traefik.enable":                                                "true",
				"traefik.swarm.network":                                         ingressNetworkName,
				"traefik.http.routers." + router + ".rule":                      "Host(`" + spec.Domain + "`)",
				"traefik.http.routers." + router + ".entrypoints":               "websecure",
				"traefik.http.routers." + router + ".tls":                       "true",
				"traefik.http.routers." + router + ".tls.certresolver":          "letsencrypt",
				"traefik.http.services." + router + ".loadbalancer.server.port": strconv.FormatUint(uint64(spec.Port), 10),
			},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{Image: image},
			RestartPolicy: &swarm.RestartPolicy{
				Condition:   swarm.RestartPolicyConditionOnFailure,
				Delay:       &delay,
				MaxAttempts: &maxAttempts,
			},
			Networks: []swarm.NetworkAttachmentConfig{{Target: networkID}},
		},
		Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
		UpdateConfig: &swarm.UpdateConfig{
			Parallelism:     1,
			FailureAction:   swarm.UpdateFailureActionRollback,
			Monitor:         30 * time.Second,
			MaxFailureRatio: 0,
			Order:           swarm.UpdateOrderStartFirst,
		},
		RollbackConfig: &swarm.UpdateConfig{
			Parallelism:   1,
			FailureAction: swarm.UpdateFailureActionPause,
			Monitor:       30 * time.Second,
			Order:         swarm.UpdateOrderStopFirst,
		},
	}
}
