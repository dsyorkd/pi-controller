package models

import (
	"time"

	"gorm.io/gorm"
)

// Node represents a Raspberry Pi node in the cluster
type Node struct {
	ID         uint           `json:"id" gorm:"primarykey"`
	Name       string         `json:"name" gorm:"uniqueIndex;not null"`
	IPAddress  string         `json:"ip_address" gorm:"not null"`
	MACAddress string         `json:"mac_address" gorm:"uniqueIndex"`
	Status     NodeStatus     `json:"status" gorm:"default:'discovered'"`
	Role       NodeRole       `json:"role" gorm:"default:'worker'"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Discovery Information
	DiscoveryMethod   DiscoveryMethod `json:"discovery_method" gorm:"not null;default:'manual'"`
	DiscoveredAt      time.Time       `json:"discovered_at"`
	NodeType          NodeType        `json:"node_type" gorm:"not null;default:'generic'"`
	ControllerVersion string          `json:"controller_version"` // Version of pi-controller if it's a controller node
	AgentPort         int             `json:"agent_port"`         // gRPC port for pi-agent (if running)

	// Hardware Information
	Architecture string `json:"architecture"`
	Model        string `json:"model"`
	SerialNumber string `json:"serial_number"`
	CPUCores     int    `json:"cpu_cores"`
	Memory       int64  `json:"memory"` // Memory in bytes

	// Kubernetes Information
	ClusterID   *uint  `json:"cluster_id,omitempty"`
	KubeVersion string `json:"kube_version"`
	NodeName    string `json:"node_name"` // Kubernetes node name

	// System Information
	OSVersion     string    `json:"os_version"`
	KernelVersion string    `json:"kernel_version"`
	LastSeen      time.Time `json:"last_seen"`

	// Relationships
	Cluster     *Cluster     `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	GPIODevices []GPIODevice `json:"gpio_devices,omitempty" gorm:"foreignKey:NodeID"`
}

// NodeStatus defines the possible states of a node
type NodeStatus string

const (
	NodeStatusDiscovered   NodeStatus = "discovered"
	NodeStatusProvisioning NodeStatus = "provisioning"
	NodeStatusReady        NodeStatus = "ready"
	NodeStatusNotReady     NodeStatus = "not_ready"
	NodeStatusMaintenance  NodeStatus = "maintenance"
	NodeStatusFailed       NodeStatus = "failed"
	NodeStatusUnknown      NodeStatus = "unknown"
)

// NodeRole defines the role of a node in the cluster
type NodeRole string

const (
	NodeRoleMaster NodeRole = "master"
	NodeRoleWorker NodeRole = "worker"
)

// DiscoveryMethod defines how a node was discovered
type DiscoveryMethod string

const (
	// DiscoveryMethodMDNS - Node discovered via mDNS service discovery
	DiscoveryMethodMDNS DiscoveryMethod = "mdns"

	// DiscoveryMethodDHCP - Node discovered via DHCP lease scanning
	DiscoveryMethodDHCP DiscoveryMethod = "dhcp"

	// DiscoveryMethodNetworkScan - Node discovered via network scanning (nmap, etc.)
	DiscoveryMethodNetworkScan DiscoveryMethod = "network_scan"

	// DiscoveryMethodManual - Node manually added by user
	DiscoveryMethodManual DiscoveryMethod = "manual"

	// DiscoveryMethodRaftCluster - Node discovered via Raft cluster membership
	DiscoveryMethodRaftCluster DiscoveryMethod = "raft_cluster"

	// DiscoveryMethodAPI - Node registered via API
	DiscoveryMethodAPI DiscoveryMethod = "api"
)

// NodeType defines the type/capability of a node
type NodeType string

const (
	// NodeTypeController - Raspberry Pi running pi-controller as a control plane
	NodeTypeController NodeType = "controller"

	// NodeTypeAgent - Raspberry Pi running pi-agent for GPIO/hardware control
	NodeTypeAgent NodeType = "agent"

	// NodeTypeGeneric - Raspberry Pi running pi-controller binary
	NodeTypeGeneric NodeType = "generic"

	// NodeTypeUnknown - Type not yet determined
	NodeTypeUnknown NodeType = "unknown"
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

// IsDiscoveredAutomatically returns true if the node was discovered automatically
func (n *Node) IsDiscoveredAutomatically() bool {
	return n.DiscoveryMethod == DiscoveryMethodMDNS ||
		n.DiscoveryMethod == DiscoveryMethodDHCP ||
		n.DiscoveryMethod == DiscoveryMethodNetworkScan ||
		n.DiscoveryMethod == DiscoveryMethodRaftCluster
}

// CanJoinRaftCluster returns true if the node can join the Raft cluster
// All nodes running pi-controller can join the cluster (NodeTypeGeneric)
func (n *Node) CanJoinRaftCluster() bool {
	return n.NodeType == NodeTypeGeneric && n.Status != NodeStatusFailed
}

// TableName returns the table name for the Node model
func (Node) TableName() string {
	return "nodes"
}
