package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNode_IsReady(t *testing.T) {
	tests := []struct {
		name   string
		status NodeStatus
		want   bool
	}{
		{
			name:   "ready node",
			status: NodeStatusReady,
			want:   true,
		},
		{
			name:   "not ready node",
			status: NodeStatusNotReady,
			want:   false,
		},
		{
			name:   "discovered node",
			status: NodeStatusDiscovered,
			want:   false,
		},
		{
			name:   "provisioning node",
			status: NodeStatusProvisioning,
			want:   false,
		},
		{
			name:   "maintenance node",
			status: NodeStatusMaintenance,
			want:   false,
		},
		{
			name:   "failed node",
			status: NodeStatusFailed,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &Node{Status: tt.status}
			assert.Equal(t, tt.want, node.IsReady())
		})
	}
}

func TestNode_IsHealthy(t *testing.T) {
	tests := []struct {
		name   string
		status NodeStatus
		want   bool
	}{
		{
			name:   "ready is healthy",
			status: NodeStatusReady,
			want:   true,
		},
		{
			name:   "maintenance is healthy",
			status: NodeStatusMaintenance,
			want:   true,
		},
		{
			name:   "not ready is not healthy",
			status: NodeStatusNotReady,
			want:   false,
		},
		{
			name:   "failed is not healthy",
			status: NodeStatusFailed,
			want:   false,
		},
		{
			name:   "discovered is not healthy",
			status: NodeStatusDiscovered,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &Node{Status: tt.status}
			assert.Equal(t, tt.want, node.IsHealthy())
		})
	}
}

func TestNode_IsMaster(t *testing.T) {
	tests := []struct {
		name string
		role NodeRole
		want bool
	}{
		{
			name: "master node",
			role: NodeRoleMaster,
			want: true,
		},
		{
			name: "worker node",
			role: NodeRoleWorker,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &Node{Role: tt.role}
			assert.Equal(t, tt.want, node.IsMaster())
		})
	}
}

func TestNode_UpdateLastSeen(t *testing.T) {
	node := &Node{
		LastSeen: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	beforeUpdate := time.Now()
	node.UpdateLastSeen()
	afterUpdate := time.Now()

	// LastSeen should be updated to current time
	assert.True(t, node.LastSeen.After(beforeUpdate) || node.LastSeen.Equal(beforeUpdate))
	assert.True(t, node.LastSeen.Before(afterUpdate) || node.LastSeen.Equal(afterUpdate))
}

func TestNodeStatus_Constants(t *testing.T) {
	// Test that all status constants are defined and have correct values
	assert.Equal(t, NodeStatus("discovered"), NodeStatusDiscovered)
	assert.Equal(t, NodeStatus("provisioning"), NodeStatusProvisioning)
	assert.Equal(t, NodeStatus("ready"), NodeStatusReady)
	assert.Equal(t, NodeStatus("not_ready"), NodeStatusNotReady)
	assert.Equal(t, NodeStatus("maintenance"), NodeStatusMaintenance)
	assert.Equal(t, NodeStatus("failed"), NodeStatusFailed)
	assert.Equal(t, NodeStatus("unknown"), NodeStatusUnknown)
}

func TestNodeRole_Constants(t *testing.T) {
	// Test that all role constants are defined and have correct values
	assert.Equal(t, NodeRole("master"), NodeRoleMaster)
	assert.Equal(t, NodeRole("worker"), NodeRoleWorker)
}

func TestNode_Relationships(t *testing.T) {
	t.Run("node with cluster", func(t *testing.T) {
		clusterID := uint(5)
		node := &Node{
			ID:        1,
			Name:      "test-node",
			ClusterID: &clusterID,
		}

		assert.NotNil(t, node.ClusterID)
		assert.Equal(t, uint(5), *node.ClusterID)
	})

	t.Run("node without cluster", func(t *testing.T) {
		node := &Node{
			ID:        1,
			Name:      "test-node",
			ClusterID: nil,
		}

		assert.Nil(t, node.ClusterID)
	})
}

func TestNode_HardwareInfo(t *testing.T) {
	node := &Node{
		Architecture: "arm64",
		Model:        "Raspberry Pi 4 Model B",
		SerialNumber: "10000000abcdef01",
		CPUCores:     4,
		Memory:       8589934592, // 8GB in bytes
	}

	assert.Equal(t, "arm64", node.Architecture)
	assert.Equal(t, "Raspberry Pi 4 Model B", node.Model)
	assert.Equal(t, "10000000abcdef01", node.SerialNumber)
	assert.Equal(t, 4, node.CPUCores)
	assert.Equal(t, int64(8589934592), node.Memory)
}

func TestNode_KubernetesInfo(t *testing.T) {
	clusterID := uint(1)
	node := &Node{
		ClusterID:   &clusterID,
		KubeVersion: "v1.28.0",
		NodeName:    "pi-node-01",
	}

	assert.NotNil(t, node.ClusterID)
	assert.Equal(t, "v1.28.0", node.KubeVersion)
	assert.Equal(t, "pi-node-01", node.NodeName)
}

func TestNode_SystemInfo(t *testing.T) {
	now := time.Now()
	node := &Node{
		OSVersion:     "Debian GNU/Linux 11 (bullseye)",
		KernelVersion: "6.1.21-v8+",
		LastSeen:      now,
	}

	assert.Equal(t, "Debian GNU/Linux 11 (bullseye)", node.OSVersion)
	assert.Equal(t, "6.1.21-v8+", node.KernelVersion)
	assert.Equal(t, now, node.LastSeen)
}
