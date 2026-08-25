// SPDX-License-Identifier: AGPL-3.0-only

package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/nectarops/nectar/internal/domain"
	"github.com/nectarops/nectar/internal/security"
)

const (
	defaultEnrollmentTTL = 30 * time.Minute
	maxEnrollmentHistory = 50
)

var (
	enrollmentIDPattern = regexp.MustCompile(`^ne_[A-Za-z0-9_-]{20,80}$`)
	hostnamePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,252}$`)
	machineIDPattern    = regexp.MustCompile(`^[a-f0-9]{64}$`)
	nodeIDPattern       = regexp.MustCompile(`^[A-Za-z0-9_-]{20,80}$`)
	architecturePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,32}$`)
	dockerEnginePattern = regexp.MustCompile(`^[0-9][0-9A-Za-z.+:~_-]{0,127}$`)
)

type NodeEnrollmentStore interface {
	CreateNodeEnrollment(context.Context, domain.NodeEnrollment, []byte) error
	NodeEnrollmentByID(context.Context, string) (domain.NodeEnrollment, error)
	NodeEnrollmentByTokenHash(context.Context, []byte) (domain.NodeEnrollment, error)
	ListNodeEnrollments(context.Context, int) ([]domain.NodeEnrollment, error)
	ClaimNodeEnrollment(
		context.Context,
		string,
		domain.NodeEnrollmentClaim,
		time.Time,
	) (domain.NodeEnrollment, error)
	RecordNodeEnrollmentProgress(
		context.Context,
		string,
		domain.NodeEnrollmentStatus,
		string,
		string,
		time.Time,
	) error
	FinishNodeEnrollment(
		context.Context,
		string,
		domain.NodeEnrollmentStatus,
		string,
		string,
		string,
		time.Time,
	) (domain.NodeEnrollment, error)
	RevokeNodeEnrollment(context.Context, string, time.Time) (domain.NodeEnrollment, error)
	NodeEnrollmentEvents(context.Context, string, int64) ([]domain.NodeEnrollmentEvent, error)
}

type NodeCluster interface {
	ListNodes(context.Context) ([]domain.SwarmNode, error)
	WorkerJoinConfiguration(context.Context) (string, string, string, error)
	Node(context.Context, string) (domain.SwarmNode, error)
	PromoteNode(context.Context, string) (domain.SwarmNode, error)
}

type EnrollmentCredential struct {
	Enrollment domain.NodeEnrollment
	Token      string
}

type NodeBootstrap struct {
	Enrollment          domain.NodeEnrollment
	DockerTargetVersion string
	ManagerAddress      string
	SwarmClusterID      string
	WorkerJoinToken     string
}

type NodeEnrollmentService struct {
	store        NodeEnrollmentStore
	cluster      NodeCluster
	policyReader DockerVersionPolicyReader
	now          func() time.Time
	ttl          time.Duration
}

func NewNodeEnrollmentService(
	store NodeEnrollmentStore,
	cluster NodeCluster,
	policyReader DockerVersionPolicyReader,
) (*NodeEnrollmentService, error) {
	if store == nil {
		return nil, errors.New("node enrollment store is required")
	}
	if cluster == nil {
		return nil, errors.New("node cluster adapter is required")
	}
	if policyReader == nil {
		return nil, errors.New("Docker version policy reader is required")
	}
	return &NodeEnrollmentService{
		store:        store,
		cluster:      cluster,
		policyReader: policyReader,
		now:          func() time.Time { return time.Now().UTC() },
		ttl:          defaultEnrollmentTTL,
	}, nil
}

func (s *NodeEnrollmentService) Create(
	ctx context.Context,
	createdBy int64,
	role domain.NodeRole,
) (EnrollmentCredential, error) {
	if createdBy <= 0 {
		return EnrollmentCredential{}, domain.ErrInvalidEnrollment
	}
	if role != domain.NodeRoleWorker && role != domain.NodeRoleManager {
		return EnrollmentCredential{}, domain.ErrInvalidEnrollment
	}

	randomID, err := security.NewToken()
	if err != nil {
		return EnrollmentCredential{}, fmt.Errorf("generate enrollment ID: %w", err)
	}
	token, err := security.NewToken()
	if err != nil {
		return EnrollmentCredential{}, fmt.Errorf("generate enrollment token: %w", err)
	}
	now := s.now()
	enrollment := domain.NodeEnrollment{
		ID:            "ne_" + randomID,
		RequestedRole: role,
		Status:        domain.NodeEnrollmentPending,
		Message:       "Enrollment command created",
		ExpiresAt:     now.Add(s.ttl),
		CreatedBy:     createdBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	tokenHash := security.HashToken(token)
	if err := s.store.CreateNodeEnrollment(ctx, enrollment, tokenHash[:]); err != nil {
		return EnrollmentCredential{}, err
	}
	return EnrollmentCredential{Enrollment: enrollment, Token: token}, nil
}

func (s *NodeEnrollmentService) List(
	ctx context.Context,
) ([]domain.NodeEnrollment, error) {
	enrollments, err := s.store.ListNodeEnrollments(ctx, maxEnrollmentHistory)
	if err != nil {
		return nil, err
	}
	for index := range enrollments {
		enrollments[index] = s.withExpirationStatus(enrollments[index])
	}
	return enrollments, nil
}

func (s *NodeEnrollmentService) Enrollment(
	ctx context.Context,
	id string,
) (domain.NodeEnrollment, error) {
	if !enrollmentIDPattern.MatchString(id) {
		return domain.NodeEnrollment{}, domain.ErrInvalidEnrollment
	}
	enrollment, err := s.store.NodeEnrollmentByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NodeEnrollment{}, domain.ErrInvalidEnrollment
	}
	return s.withExpirationStatus(enrollment), err
}

func (s *NodeEnrollmentService) Events(
	ctx context.Context,
	id string,
	afterID int64,
) ([]domain.NodeEnrollmentEvent, error) {
	if _, err := s.Enrollment(ctx, id); err != nil {
		return nil, err
	}
	return s.store.NodeEnrollmentEvents(ctx, id, afterID)
}

func (s *NodeEnrollmentService) Revoke(
	ctx context.Context,
	id string,
) (domain.NodeEnrollment, error) {
	if !enrollmentIDPattern.MatchString(id) {
		return domain.NodeEnrollment{}, domain.ErrInvalidEnrollment
	}
	enrollment, err := s.Enrollment(ctx, id)
	if err != nil {
		return domain.NodeEnrollment{}, err
	}
	if enrollment.Terminal() {
		return domain.NodeEnrollment{}, domain.ErrEnrollmentTerminal
	}
	return s.store.RevokeNodeEnrollment(ctx, id, s.now())
}

func (s *NodeEnrollmentService) Nodes(
	ctx context.Context,
) ([]domain.SwarmNode, error) {
	nodes, err := s.cluster.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	targetVersion, err := s.policyReader.DesiredDockerVersion(ctx)
	if err != nil {
		return nil, err
	}
	for index := range nodes {
		nodes[index].DesiredDockerVersion = targetVersion
		nodes[index].VersionDrift = targetVersion != "" && nodes[index].DockerVersion != targetVersion
	}
	return nodes, nil
}

func (s *NodeEnrollmentService) Claim(
	ctx context.Context,
	token string,
	claim domain.NodeEnrollmentClaim,
) (NodeBootstrap, error) {
	if err := validateNodeEnrollmentClaim(claim); err != nil {
		return NodeBootstrap{}, err
	}
	enrollment, err := s.authenticatedEnrollment(ctx, token, claim.MachineIDHash)
	if err != nil {
		return NodeBootstrap{}, err
	}
	enrollment, err = s.store.ClaimNodeEnrollment(ctx, enrollment.ID, claim, s.now())
	if err != nil {
		return NodeBootstrap{}, err
	}
	targetVersion, err := s.policyReader.DesiredDockerVersion(ctx)
	if err != nil {
		return NodeBootstrap{}, err
	}
	if targetVersion == "" {
		return NodeBootstrap{}, errors.New("cluster Docker target version is not configured")
	}
	managerAddress, clusterID, workerToken, err := s.cluster.WorkerJoinConfiguration(ctx)
	if err != nil {
		return NodeBootstrap{}, err
	}
	return NodeBootstrap{
		Enrollment:          enrollment,
		DockerTargetVersion: targetVersion,
		ManagerAddress:      managerAddress,
		SwarmClusterID:      clusterID,
		WorkerJoinToken:     workerToken,
	}, nil
}

func (s *NodeEnrollmentService) Progress(
	ctx context.Context,
	token string,
	machineIDHash string,
	phase string,
	dockerVersion string,
) error {
	enrollment, err := s.authenticatedEnrollment(ctx, token, machineIDHash)
	if err != nil {
		return err
	}
	status, message, ok := enrollmentProgress(phase)
	if !ok {
		return domain.ErrInvalidEnrollment
	}
	if !validEnrollmentProgressTransition(enrollment.Status, status) {
		return domain.ErrInvalidEnrollment
	}
	if dockerVersion != "" && !dockerEnginePattern.MatchString(dockerVersion) {
		return domain.ErrInvalidEnrollment
	}
	return s.store.RecordNodeEnrollmentProgress(
		ctx,
		enrollment.ID,
		status,
		message,
		dockerVersion,
		s.now(),
	)
}

func (s *NodeEnrollmentService) Complete(
	ctx context.Context,
	token string,
	machineIDHash string,
	nodeID string,
) (domain.NodeEnrollment, error) {
	enrollment, err := s.boundEnrollment(ctx, token, machineIDHash)
	if err != nil {
		return domain.NodeEnrollment{}, err
	}
	switch enrollment.Status {
	case domain.NodeEnrollmentCompleted:
		return enrollment, nil
	case domain.NodeEnrollmentPromotionBlocked:
		return enrollment, domain.ErrManagerVersionMismatch
	case domain.NodeEnrollmentRevoked, domain.NodeEnrollmentExpired:
		return domain.NodeEnrollment{}, domain.ErrEnrollmentTerminal
	}
	if !nodeIDPattern.MatchString(nodeID) {
		return domain.NodeEnrollment{}, domain.ErrInvalidEnrollment
	}
	if enrollment.Status != domain.NodeEnrollmentVerifying {
		return domain.NodeEnrollment{}, domain.ErrInvalidEnrollment
	}

	node, err := s.cluster.Node(ctx, nodeID)
	if err != nil {
		return domain.NodeEnrollment{}, err
	}
	if node.Status != "ready" {
		return domain.NodeEnrollment{}, domain.ErrNodeNotReady
	}
	if !strings.EqualFold(node.Hostname, enrollment.Hostname) {
		return domain.NodeEnrollment{}, errors.New("joined Swarm node hostname does not match the claimed host")
	}

	now := s.now()
	if enrollment.RequestedRole == domain.NodeRoleManager {
		targetVersion, policyErr := s.policyReader.DesiredDockerVersion(ctx)
		if policyErr != nil {
			return domain.NodeEnrollment{}, policyErr
		}
		if node.DockerVersion != targetVersion {
			message := fmt.Sprintf(
				"Joined as Worker; Manager promotion requires Docker %s, found %s",
				targetVersion,
				node.DockerVersion,
			)
			finished, finishErr := s.store.FinishNodeEnrollment(
				ctx,
				enrollment.ID,
				domain.NodeEnrollmentPromotionBlocked,
				node.ID,
				node.DockerVersion,
				message,
				now,
			)
			if finishErr != nil {
				return domain.NodeEnrollment{}, finishErr
			}
			return finished, domain.ErrManagerVersionMismatch
		}

		if err := s.store.RecordNodeEnrollmentProgress(
			ctx,
			enrollment.ID,
			domain.NodeEnrollmentPromoting,
			"Node is Ready; promoting it to Manager",
			node.DockerVersion,
			now,
		); err != nil {
			return domain.NodeEnrollment{}, err
		}
		node, err = s.cluster.PromoteNode(ctx, node.ID)
		if err != nil {
			_ = s.store.RecordNodeEnrollmentProgress(
				ctx,
				enrollment.ID,
				domain.NodeEnrollmentFailed,
				"Manager promotion failed; rerun the bound enrollment command to retry",
				node.DockerVersion,
				s.now(),
			)
			return domain.NodeEnrollment{}, err
		}
	}

	message := "Node joined the Swarm as Worker"
	if node.Role == domain.NodeRoleManager {
		message = "Node joined the Swarm and is Reachable as Manager"
	}
	return s.store.FinishNodeEnrollment(
		ctx,
		enrollment.ID,
		domain.NodeEnrollmentCompleted,
		node.ID,
		node.DockerVersion,
		message,
		s.now(),
	)
}

func (s *NodeEnrollmentService) authenticatedEnrollment(
	ctx context.Context,
	token string,
	machineIDHash string,
) (domain.NodeEnrollment, error) {
	enrollment, err := s.boundEnrollment(ctx, token, machineIDHash)
	if err != nil {
		return domain.NodeEnrollment{}, err
	}
	if enrollment.Terminal() {
		return domain.NodeEnrollment{}, domain.ErrEnrollmentTerminal
	}
	return enrollment, nil
}

func (s *NodeEnrollmentService) boundEnrollment(
	ctx context.Context,
	token string,
	machineIDHash string,
) (domain.NodeEnrollment, error) {
	if len(token) < 32 || len(token) > 128 {
		return domain.NodeEnrollment{}, domain.ErrInvalidEnrollmentToken
	}
	tokenHash := security.HashToken(token)
	enrollment, err := s.store.NodeEnrollmentByTokenHash(ctx, tokenHash[:])
	if err != nil {
		return domain.NodeEnrollment{}, err
	}
	if !s.now().Before(enrollment.ExpiresAt) {
		return domain.NodeEnrollment{}, domain.ErrEnrollmentExpired
	}
	if enrollment.MachineIDHash != "" && enrollment.MachineIDHash != machineIDHash {
		return domain.NodeEnrollment{}, domain.ErrEnrollmentClaimed
	}
	return enrollment, nil
}

func (s *NodeEnrollmentService) withExpirationStatus(
	enrollment domain.NodeEnrollment,
) domain.NodeEnrollment {
	if !enrollment.Terminal() && !s.now().Before(enrollment.ExpiresAt) {
		enrollment.Status = domain.NodeEnrollmentExpired
		enrollment.Message = "Enrollment command expired before the node joined"
	}
	return enrollment
}

func validateNodeEnrollmentClaim(claim domain.NodeEnrollmentClaim) error {
	if !hostnamePattern.MatchString(claim.Hostname) {
		return domain.ErrInvalidEnrollment
	}
	if !machineIDPattern.MatchString(claim.MachineIDHash) {
		return domain.ErrInvalidEnrollment
	}
	if strings.TrimSpace(claim.OperatingSystem) == "" || len(claim.OperatingSystem) > 160 {
		return domain.ErrInvalidEnrollment
	}
	if !architecturePattern.MatchString(claim.Architecture) {
		return domain.ErrInvalidEnrollment
	}
	if net.ParseIP(claim.AdvertiseAddress) == nil || net.ParseIP(claim.DataPathAddress) == nil {
		return domain.ErrInvalidEnrollment
	}
	if claim.DockerVersion != "" && !dockerEnginePattern.MatchString(claim.DockerVersion) {
		return domain.ErrInvalidEnrollment
	}
	return nil
}

func enrollmentProgress(
	phase string,
) (domain.NodeEnrollmentStatus, string, bool) {
	switch phase {
	case "installing":
		return domain.NodeEnrollmentInstalling, "Installing the cluster Docker target version", true
	case "docker-ready":
		return domain.NodeEnrollmentInstalling, "Docker Engine is healthy", true
	case "joining":
		return domain.NodeEnrollmentJoining, "Joining the Docker Swarm as Worker", true
	case "verifying":
		return domain.NodeEnrollmentVerifying, "Waiting for the Swarm Manager to verify the node", true
	case "failed":
		return domain.NodeEnrollmentFailed, "Node enrollment failed; inspect the client diagnostic", true
	default:
		return "", "", false
	}
}

func validEnrollmentProgressTransition(
	current domain.NodeEnrollmentStatus,
	next domain.NodeEnrollmentStatus,
) bool {
	if next == domain.NodeEnrollmentFailed {
		switch current {
		case domain.NodeEnrollmentClaimed,
			domain.NodeEnrollmentInstalling,
			domain.NodeEnrollmentJoining,
			domain.NodeEnrollmentVerifying,
			domain.NodeEnrollmentFailed:
			return true
		default:
			return false
		}
	}

	switch next {
	case domain.NodeEnrollmentInstalling:
		return current == domain.NodeEnrollmentClaimed || current == domain.NodeEnrollmentInstalling
	case domain.NodeEnrollmentJoining:
		return current == domain.NodeEnrollmentInstalling || current == domain.NodeEnrollmentJoining
	case domain.NodeEnrollmentVerifying:
		return current == domain.NodeEnrollmentInstalling ||
			current == domain.NodeEnrollmentJoining ||
			current == domain.NodeEnrollmentVerifying
	default:
		return false
	}
}
