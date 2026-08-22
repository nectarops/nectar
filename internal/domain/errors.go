// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "errors"

var (
	ErrAlreadyConfigured     = errors.New("Nectar is already configured")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrInvalidInitToken      = errors.New("invalid initialization token")
	ErrInitTokenMissing      = errors.New("initialization token is not configured")
	ErrDockerVersionConflict = errors.New("desired Docker version conflicts with the recorded cluster policy")
	ErrUnauthenticated       = errors.New("authentication required")
)
