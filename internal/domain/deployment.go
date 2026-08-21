// SPDX-License-Identifier: AGPL-3.0-only

package domain

type DeploymentSpec struct {
	ServiceName string `json:"serviceName"`
	Image       string `json:"image"`
	Version     string `json:"version"`
	Domain      string `json:"domain"`
	ACMEEmail   string `json:"acmeEmail"`
	Port        uint32 `json:"port"`
	Replicas    uint64 `json:"replicas"`
}

type DeploymentResult struct {
	ServiceID string   `json:"serviceId"`
	Image     string   `json:"image"`
	Updated   bool     `json:"updated"`
	Warnings  []string `json:"warnings"`
}
