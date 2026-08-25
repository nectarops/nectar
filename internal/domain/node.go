// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

type NodeRole string

const (
	NodeRoleWorker  NodeRole = "worker"
	NodeRoleManager NodeRole = "manager"
)

type NodeEnrollmentStatus string

const (
	NodeEnrollmentPending          NodeEnrollmentStatus = "pending"
	NodeEnrollmentClaimed          NodeEnrollmentStatus = "claimed"
	NodeEnrollmentInstalling       NodeEnrollmentStatus = "installing"
	NodeEnrollmentJoining          NodeEnrollmentStatus = "joining"
	NodeEnrollmentVerifying        NodeEnrollmentStatus = "verifying"
	NodeEnrollmentPromoting        NodeEnrollmentStatus = "promoting"
	NodeEnrollmentCompleted        NodeEnrollmentStatus = "completed"
	NodeEnrollmentPromotionBlocked NodeEnrollmentStatus = "promotion_blocked"
	NodeEnrollmentFailed           NodeEnrollmentStatus = "failed"
	NodeEnrollmentRevoked          NodeEnrollmentStatus = "revoked"
	NodeEnrollmentExpired          NodeEnrollmentStatus = "expired"
)

type NodeEnrollment struct {
	ID               string               `json:"id"`
	RequestedRole    NodeRole             `json:"requestedRole"`
	Status           NodeEnrollmentStatus `json:"status"`
	Hostname         string               `json:"hostname,omitempty"`
	MachineIDHash    string               `json:"-"`
	OperatingSystem  string               `json:"operatingSystem,omitempty"`
	Architecture     string               `json:"architecture,omitempty"`
	AdvertiseAddress string               `json:"advertiseAddress,omitempty"`
	DataPathAddress  string               `json:"dataPathAddress,omitempty"`
	DockerVersion    string               `json:"dockerVersion,omitempty"`
	NodeID           string               `json:"nodeId,omitempty"`
	Message          string               `json:"message,omitempty"`
	ExpiresAt        time.Time            `json:"expiresAt"`
	CreatedBy        int64                `json:"createdBy"`
	CreatedAt        time.Time            `json:"createdAt"`
	UpdatedAt        time.Time            `json:"updatedAt"`
}

func (e NodeEnrollment) Terminal() bool {
	switch e.Status {
	case NodeEnrollmentCompleted,
		NodeEnrollmentPromotionBlocked,
		NodeEnrollmentRevoked,
		NodeEnrollmentExpired:
		return true
	default:
		return false
	}
}

type NodeEnrollmentClaim struct {
	Hostname         string
	MachineIDHash    string
	OperatingSystem  string
	Architecture     string
	AdvertiseAddress string
	DataPathAddress  string
	DockerVersion    string
}

type NodeEnrollmentEvent struct {
	ID           int64                `json:"id"`
	EnrollmentID string               `json:"enrollmentId"`
	Status       NodeEnrollmentStatus `json:"status"`
	Message      string               `json:"message"`
	CreatedAt    time.Time            `json:"createdAt"`
}

type SwarmNode struct {
	ID                   string   `json:"id"`
	Hostname             string   `json:"hostname"`
	Role                 NodeRole `json:"role"`
	Status               string   `json:"status"`
	Availability         string   `json:"availability"`
	ManagerStatus        string   `json:"managerStatus,omitempty"`
	Address              string   `json:"address"`
	ManagerAddress       string   `json:"managerAddress,omitempty"`
	OperatingSystem      string   `json:"operatingSystem"`
	Architecture         string   `json:"architecture"`
	DockerVersion        string   `json:"dockerVersion"`
	DesiredDockerVersion string   `json:"desiredDockerVersion"`
	VersionDrift         bool     `json:"versionDrift"`
}
