// SPDX-License-Identifier: AGPL-3.0-only

package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func NewToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(value), nil
}

func HashToken(token string) [sha256.Size]byte {
	return sha256.Sum256([]byte(token))
}
