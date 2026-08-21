// SPDX-License-Identifier: AGPL-3.0-only

package domain

type ClusterSnapshot struct {
	Available         bool   `json:"available"`
	Error             string `json:"error,omitempty"`
	Hostname          string `json:"hostname"`
	OperatingSystem   string `json:"operatingSystem"`
	Architecture      string `json:"architecture"`
	KernelVersion     string `json:"kernelVersion"`
	DockerVersion     string `json:"dockerVersion"`
	DockerAPIVersion  string `json:"dockerApiVersion"`
	SwarmState        string `json:"swarmState"`
	NodeID            string `json:"nodeId,omitempty"`
	NodeRole          string `json:"nodeRole"`
	NodeStatus        string `json:"nodeStatus"`
	Availability      string `json:"availability"`
	ManagerStatus     string `json:"managerStatus,omitempty"`
	Managers          int    `json:"managers"`
	Nodes             int    `json:"nodes"`
	CPUs              int    `json:"cpus"`
	MemoryBytes       int64  `json:"memoryBytes"`
	ContainersRunning int    `json:"containersRunning"`
	Images            int    `json:"images"`
}
