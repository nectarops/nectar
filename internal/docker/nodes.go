// SPDX-License-Identifier: AGPL-3.0-only

package docker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/nectarops/nectar/internal/domain"
)

func (i *Inspector) ListNodes(ctx context.Context) ([]domain.SwarmNode, error) {
	requestCtx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	result, err := i.client.NodeList(requestCtx, client.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list Swarm nodes: %w", err)
	}
	nodes := make([]domain.SwarmNode, 0, len(result.Items))
	for _, node := range result.Items {
		nodes = append(nodes, swarmNode(node))
	}
	return nodes, nil
}

func (i *Inspector) WorkerJoinConfiguration(
	ctx context.Context,
) (string, string, string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	infoResult, err := i.client.Info(requestCtx, client.InfoOptions{})
	if err != nil {
		return "", "", "", fmt.Errorf("read Docker Engine info: %w", err)
	}
	info := infoResult.Info
	if info.Swarm.LocalNodeState != swarm.LocalNodeStateActive || !info.Swarm.ControlAvailable {
		return "", "", "", errors.New("Docker Engine is not an active Swarm Manager")
	}

	managerAddress := ""
	nodes, err := i.client.NodeList(requestCtx, client.NodeListOptions{})
	if err != nil {
		return "", "", "", fmt.Errorf("list Swarm managers: %w", err)
	}
	for _, node := range nodes.Items {
		if node.ID == info.Swarm.NodeID && node.ManagerStatus != nil {
			managerAddress = node.ManagerStatus.Addr
			break
		}
	}
	if managerAddress == "" && info.Swarm.NodeAddr != "" {
		managerAddress = net.JoinHostPort(info.Swarm.NodeAddr, "2377")
	}
	if managerAddress == "" {
		return "", "", "", errors.New("Swarm Manager advertise address is unavailable")
	}
	if info.Swarm.Cluster == nil || info.Swarm.Cluster.ID == "" {
		return "", "", "", errors.New("Swarm cluster ID is unavailable")
	}

	inspected, err := i.client.SwarmInspect(requestCtx, client.SwarmInspectOptions{})
	if err != nil {
		return "", "", "", fmt.Errorf("inspect Swarm join configuration: %w", err)
	}
	if inspected.Swarm.JoinTokens.Worker == "" {
		return "", "", "", errors.New("Swarm Worker join token is unavailable")
	}
	return managerAddress, info.Swarm.Cluster.ID, inspected.Swarm.JoinTokens.Worker, nil
}

func (i *Inspector) Node(
	ctx context.Context,
	nodeID string,
) (domain.SwarmNode, error) {
	requestCtx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	result, err := i.client.NodeInspect(requestCtx, nodeID, client.NodeInspectOptions{})
	if err != nil {
		return domain.SwarmNode{}, fmt.Errorf("inspect Swarm node: %w", err)
	}
	return swarmNode(result.Node), nil
}

func (i *Inspector) PromoteNode(
	ctx context.Context,
	nodeID string,
) (domain.SwarmNode, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := i.client.NodeInspect(requestCtx, nodeID, client.NodeInspectOptions{})
	if err != nil {
		return domain.SwarmNode{}, fmt.Errorf("inspect Worker before promotion: %w", err)
	}
	node := result.Node
	if node.Status.State != swarm.NodeStateReady {
		return domain.SwarmNode{}, domain.ErrNodeNotReady
	}
	if node.Spec.Role != swarm.NodeRoleManager {
		node.Spec.Role = swarm.NodeRoleManager
		_, err := i.client.NodeUpdate(requestCtx, node.ID, client.NodeUpdateOptions{
			Version: node.Version,
			Spec:    node.Spec,
		})
		if err != nil {
			return domain.SwarmNode{}, fmt.Errorf("promote Worker to Manager: %w", err)
		}
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		inspected, inspectErr := i.client.NodeInspect(
			requestCtx,
			nodeID,
			client.NodeInspectOptions{},
		)
		if inspectErr != nil {
			return domain.SwarmNode{}, fmt.Errorf("verify promoted Manager: %w", inspectErr)
		}
		promoted := inspected.Node.Spec.Role == swarm.NodeRoleManager
		reachable := inspected.Node.ManagerStatus != nil &&
			(inspected.Node.ManagerStatus.Reachability == swarm.ReachabilityReachable ||
				inspected.Node.ManagerStatus.Leader)
		if promoted && reachable {
			return swarmNode(inspected.Node), nil
		}
		select {
		case <-requestCtx.Done():
			return domain.SwarmNode{}, fmt.Errorf("wait for promoted Manager: %w", requestCtx.Err())
		case <-ticker.C:
		}
	}
}

func swarmNode(node swarm.Node) domain.SwarmNode {
	managerStatus := ""
	managerAddress := ""
	if node.ManagerStatus != nil {
		managerStatus = string(node.ManagerStatus.Reachability)
		managerAddress = node.ManagerStatus.Addr
		if node.ManagerStatus.Leader {
			managerStatus = "leader"
		}
	}
	return domain.SwarmNode{
		ID:              node.ID,
		Hostname:        node.Description.Hostname,
		Role:            domain.NodeRole(node.Spec.Role),
		Status:          string(node.Status.State),
		Availability:    string(node.Spec.Availability),
		ManagerStatus:   managerStatus,
		Address:         node.Status.Addr,
		ManagerAddress:  managerAddress,
		OperatingSystem: node.Description.Platform.OS,
		Architecture:    node.Description.Platform.Architecture,
		DockerVersion:   node.Description.Engine.EngineVersion,
	}
}
