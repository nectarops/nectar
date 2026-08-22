// SPDX-License-Identifier: AGPL-3.0-only

package domain

type ManagementAccess struct {
	Domain    string `json:"domain"`
	ACMEEmail string `json:"acmeEmail"`
}
