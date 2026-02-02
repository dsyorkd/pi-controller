package models

import (
	"time"

	"gorm.io/gorm"
)

// Cluster represents a Kubernetes cluster managed by pi-controller
type Cluster struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"uniqueIndex;not null"`
	Description string         `json:"description"`
	Status      ClusterStatus  `json:"status" gorm:"default:'pending'"`
	Version     string         `json:"version"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Configuration
	KubeConfig     string `json:"-" gorm:"type:text"` // Base64 encoded kubeconfig
	MasterEndpoint string `json:"master_endpoint"`

	// Relationships
	Nodes []Node `json:"nodes,omitempty" gorm:"foreignKey:ClusterID"`
}

// ClusterStatus defines the possible states of a cluster
type ClusterStatus string

const (
	ClusterStatusPending      ClusterStatus = "pending"
	ClusterStatusProvisioning ClusterStatus = "provisioning"
	ClusterStatusActive       ClusterStatus = "active"
	ClusterStatusDegraded     ClusterStatus = "degraded"
	ClusterStatusMaintenance  ClusterStatus = "maintenance"
	ClusterStatusFailed       ClusterStatus = "failed"
)

// IsActive returns true if the cluster is in an active state
func (c *Cluster) IsActive() bool {
	return c.Status == ClusterStatusActive
}

// IsHealthy returns true if the cluster is in a healthy state
func (c *Cluster) IsHealthy() bool {
	return c.Status == ClusterStatusActive || c.Status == ClusterStatusMaintenance
}

// TableName returns the table name for the Cluster model
func (Cluster) TableName() string {
	return "clusters"
}
