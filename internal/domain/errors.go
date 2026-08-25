// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

var (
	ErrAlreadyConfigured       = errors.New("Nectar is already configured")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidInitToken        = errors.New("invalid initialization token")
	ErrInitTokenMissing        = errors.New("initialization token is not configured")
	ErrInvalidManagementAccess = errors.New("invalid management access")
	ErrDockerVersionConflict   = errors.New("desired Docker version conflicts with the recorded cluster policy")
	ErrEnrollmentClaimed       = errors.New("node enrollment is already claimed by another host")
	ErrEnrollmentExpired       = errors.New("node enrollment has expired")
	ErrEnrollmentTerminal      = errors.New("node enrollment is no longer active")
	ErrInvalidEnrollment       = errors.New("invalid node enrollment")
	ErrInvalidEnrollmentToken  = errors.New("invalid node enrollment token")
	ErrManagerVersionMismatch  = errors.New("node Docker version does not match the Manager target")
	ErrNodeNotReady            = errors.New("Swarm node is not ready")
	ErrUnauthenticated         = errors.New("authentication required")
)
