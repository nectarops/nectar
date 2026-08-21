// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type StoredUser struct {
	User
	PasswordHash string
}

type Session struct {
	Token     string
	ExpiresAt time.Time
}
