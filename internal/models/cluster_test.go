package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCluster_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status ClusterStatus
		want   bool
	}{
		{
			name:   "active cluster",
			status: ClusterStatusActive,
			want:   true,
		},
		{
			name:   "pending cluster",
			status: ClusterStatusPending,
			want:   false,
		},
		{
			name:   "provisioning cluster",
			status: ClusterStatusProvisioning,
			want:   false,
		},
		{
			name:   "failed cluster",
			status: ClusterStatusFailed,
			want:   false,
		},
		{
			name:   "degraded cluster",
			status: ClusterStatusDegraded,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &Cluster{Status: tt.status}
			assert.Equal(t, tt.want, cluster.IsActive())
		})
	}
}

func TestCluster_IsHealthy(t *testing.T) {
	tests := []struct {
		name   string
		status ClusterStatus
		want   bool
	}{
		{
			name:   "active is healthy",
			status: ClusterStatusActive,
			want:   true,
		},
		{
			name:   "maintenance is healthy",
			status: ClusterStatusMaintenance,
			want:   true,
		},
		{
			name:   "degraded is not healthy",
			status: ClusterStatusDegraded,
			want:   false,
		},
		{
			name:   "failed is not healthy",
			status: ClusterStatusFailed,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &Cluster{Status: tt.status}
			assert.Equal(t, tt.want, cluster.IsHealthy())
		})
	}
}

func TestClusterStatus_Constants(t *testing.T) {
	assert.Equal(t, ClusterStatus("active"), ClusterStatusActive)
	assert.Equal(t, ClusterStatus("pending"), ClusterStatusPending)
	assert.Equal(t, ClusterStatus("provisioning"), ClusterStatusProvisioning)
	assert.Equal(t, ClusterStatus("degraded"), ClusterStatusDegraded)
	assert.Equal(t, ClusterStatus("maintenance"), ClusterStatusMaintenance)
	assert.Equal(t, ClusterStatus("failed"), ClusterStatusFailed)
}

func TestCluster_BasicFields(t *testing.T) {
	cluster := &Cluster{
		ID:             1,
		Name:           "production-cluster",
		Description:    "Production Kubernetes cluster",
		Status:         ClusterStatusActive,
		Version:        "v1.28.0",
		MasterEndpoint: "https://192.168.1.100:6443",
	}

	assert.Equal(t, uint(1), cluster.ID)
	assert.Equal(t, "production-cluster", cluster.Name)
	assert.Equal(t, "Production Kubernetes cluster", cluster.Description)
	assert.Equal(t, ClusterStatusActive, cluster.Status)
	assert.Equal(t, "v1.28.0", cluster.Version)
	assert.Equal(t, "https://192.168.1.100:6443", cluster.MasterEndpoint)
}
