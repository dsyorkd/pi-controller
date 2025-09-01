package models

import (
	"time"

	"gorm.io/gorm"
)

// Node represents a Raspberry Pi node in the cluster
type Node struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Name      string         `json:"name" gorm:"uniqueIndex;not null"`
	IPAddress string         `json:"ip_address" gorm:"not null"`
	MACAddress string        `json:"mac_address" gorm:"uniqueIndex"`
	Status    NodeStatus     `json:"status" gorm:"default:'discovered'"`
	Role      NodeRole       `json:"role" gorm:"default:'worker'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Hardware Information
	Architecture string `json:"architecture"`
	Model        string `json:"model"`
	SerialNumber string `json:"serial_number"`
	CPUCores     int    `json:"cpu_cores"`
	Memory       int64  `json:"memory"` // Memory in bytes
	
	// Kubernetes Information
	ClusterID    *uint   `json:"cluster_id,omitempty"`
	KubeVersion  string  `json:"kube_version"`
	NodeName     string  `json:"node_name"` // Kubernetes node name
	
	// System Information
	OSVersion    string `json:"os_version"`
	KernelVersion string `json:"kernel_version"`
	LastSeen     time.Time `json:"last_seen"`
	
	// Relationships
	Cluster     *Cluster     `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	GPIODevices []GPIODevice `json:"gpio_devices,omitempty" gorm:"foreignKey:NodeID"`
}

// NodeStatus defines the possible states of a node
type NodeStatus string

const (
	NodeStatusDiscovered    NodeStatus = "discovered"
	NodeStatusProvisioning  NodeStatus = "provisioning"
	NodeStatusReady         NodeStatus = "ready"
	NodeStatusNotReady      NodeStatus = "not_ready"
	NodeStatusMaintenance   NodeStatus = "maintenance"
	NodeStatusFailed        NodeStatus = "failed"
	NodeStatusUnknown       NodeStatus = "unknown"
)

// NodeRole defines the role of a node in the cluster
type NodeRole string

const (
	NodeRoleMaster NodeRole = "master"
	NodeRoleWorker NodeRole = "worker"
)

// IsReady returns true if the node is ready to accept workloads
func (n *Node) IsReady() bool {
	return n.Status == NodeStatusReady
}

// IsHealthy returns true if the node is in a healthy state
func (n *Node) IsHealthy() bool {
	return n.Status == NodeStatusReady || n.Status == NodeStatusMaintenance
}

// IsMaster returns true if the node is a master node
func (n *Node) IsMaster() bool {
	return n.Role == NodeRoleMaster
}

// UpdateLastSeen updates the last seen timestamp
func (n *Node) UpdateLastSeen() {
	n.LastSeen = time.Now()
}

// TableName returns the table name for the Node model
func (Node) TableName() string {
	return "nodes"
}