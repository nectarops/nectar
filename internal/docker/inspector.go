// SPDX-License-Identifier: AGPL-3.0-only

package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/ranen/dock-weaver/internal/domain"
)

type Inspector struct {
	client  *client.Client
	timeout time.Duration
}

func NewInspector(timeout time.Duration) (*Inspector, error) {
	apiClient, err := client.New(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create Docker client: %w", err)
	}

	return &Inspector{
		client:  apiClient,
		timeout: timeout,
	}, nil
}

func (i *Inspector) Close() error {
	return i.client.Close()
}

func (i *Inspector) Ready(ctx context.Context) error {
	requestCtx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	_, err := i.client.Ping(requestCtx, client.PingOptions{})
	if err != nil {
		return fmt.Errorf("ping Docker Engine: %w", err)
	}

	return nil
}

func (i *Inspector) Inspect(ctx context.Context) (domain.ClusterSnapshot, error) {
	requestCtx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	infoResult, err := i.client.Info(requestCtx, client.InfoOptions{})
	if err != nil {
		return domain.ClusterSnapshot{}, fmt.Errorf("read Docker Engine info: %w", err)
	}
	version, err := i.client.ServerVersion(requestCtx, client.ServerVersionOptions{})
	if err != nil {
		return domain.ClusterSnapshot{}, fmt.Errorf("read Docker Engine version: %w", err)
	}

	info := infoResult.Info
	snapshot := domain.ClusterSnapshot{
		Available:         true,
		Hostname:          info.Name,
		OperatingSystem:   info.OperatingSystem,
		Architecture:      info.Architecture,
		KernelVersion:     info.KernelVersion,
		DockerVersion:     version.Version,
		DockerAPIVersion:  version.APIVersion,
		SwarmState:        string(info.Swarm.LocalNodeState),
		NodeID:            info.Swarm.NodeID,
		NodeRole:          "standalone",
		NodeStatus:        "unknown",
		Availability:      "unknown",
		Managers:          info.Swarm.Managers,
		Nodes:             info.Swarm.Nodes,
		CPUs:              info.NCPU,
		MemoryBytes:       info.MemTotal,
		ContainersRunning: info.ContainersRunning,
		Images:            info.Images,
	}

	if info.Swarm.LocalNodeState != swarm.LocalNodeStateActive {
		return snapshot, nil
	}
	if !info.Swarm.ControlAvailable {
		snapshot.NodeRole = "worker"
		return snapshot, nil
	}

	nodes, err := i.client.NodeList(requestCtx, client.NodeListOptions{})
	if err != nil {
		return domain.ClusterSnapshot{}, fmt.Errorf("list Swarm nodes: %w", err)
	}

	for _, node := range nodes.Items {
		if node.ID != info.Swarm.NodeID {
			continue
		}

		snapshot.NodeRole = string(node.Spec.Role)
		snapshot.NodeStatus = string(node.Status.State)
		snapshot.Availability = string(node.Spec.Availability)
		if node.ManagerStatus != nil {
			snapshot.ManagerStatus = string(node.ManagerStatus.Reachability)
			if node.ManagerStatus.Leader {
				snapshot.ManagerStatus = "leader"
			}
		}
		break
	}

	return snapshot, nil
}
