package services

import (
	"testing"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupNodeTest(t *testing.T) (*NodeService, *storage.Database, func()) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)

	service := NewNodeService(db, log)

	cleanup := func() {
		db.Close()
	}

	return service, db, cleanup
}

func TestNewNodeService(t *testing.T) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)
	defer db.Close()

	service := NewNodeService(db, log)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.logger)
}

func TestNodeService_Create(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		req     CreateNodeRequest
		wantErr bool
	}{
		{
			name: "valid node creation",
			req: CreateNodeRequest{
				Name:         "test-node",
				IPAddress:    "192.168.1.100",
				MACAddress:   "aa:bb:cc:dd:ee:01",
				Architecture: "arm64",
				Role:         models.NodeRoleWorker,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: CreateNodeRequest{
				Name:       "",
				IPAddress:  "192.168.1.101",
				MACAddress: "aa:bb:cc:dd:ee:02",
			},
			wantErr: true, // Service validates that names cannot be empty
		},
		{
			name: "missing ip address",
			req: CreateNodeRequest{
				Name:       "test-node-2",
				IPAddress:  "",
				MACAddress: "aa:bb:cc:dd:ee:03",
			},
			wantErr: true, // Service validates that IP addresses cannot be empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := service.Create(tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
				assert.NotZero(t, node.ID)
				assert.Equal(t, tt.req.Name, node.Name)
				assert.Equal(t, tt.req.IPAddress, node.IPAddress)
			}
		})
	}
}

func TestNodeService_GetByID(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create a node first
	created, err := service.Create(CreateNodeRequest{
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:20",
	})
	require.NoError(t, err)

	t.Run("existing node", func(t *testing.T) {
		node, err := service.GetByID(created.ID, false)
		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, created.ID, node.ID)
		assert.Equal(t, created.Name, node.Name)
	})

	t.Run("non-existent node", func(t *testing.T) {
		node, err := service.GetByID(99999, false)
		assert.Error(t, err)
		assert.Nil(t, node)
	})
}

func TestNodeService_GetByName(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create a node first
	created, err := service.Create(CreateNodeRequest{
		Name:       "unique-node-name",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:21",
	})
	require.NoError(t, err)

	t.Run("existing node by name", func(t *testing.T) {
		node, err := service.GetByName(created.Name)
		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, created.ID, node.ID)
		assert.Equal(t, created.Name, node.Name)
	})

	t.Run("non-existent node by name", func(t *testing.T) {
		node, err := service.GetByName("non-existent-node")
		assert.Error(t, err)
		assert.Nil(t, node)
	})
}

func TestNodeService_GetByIPAddress(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create a node first
	created, err := service.Create(CreateNodeRequest{
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:22",
	})
	require.NoError(t, err)

	t.Run("existing node by IP", func(t *testing.T) {
		node, err := service.GetByIPAddress(created.IPAddress)
		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, created.ID, node.ID)
		assert.Equal(t, created.IPAddress, node.IPAddress)
	})

	t.Run("non-existent node by IP", func(t *testing.T) {
		node, err := service.GetByIPAddress("192.168.1.250")
		assert.Error(t, err)
		assert.Nil(t, node)
	})
}

func TestNodeService_List(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create multiple nodes
	service.Create(CreateNodeRequest{Name: "node-1", IPAddress: "192.168.1.101", MACAddress: "aa:bb:cc:dd:ee:10"})
	service.Create(CreateNodeRequest{Name: "node-2", IPAddress: "192.168.1.102", MACAddress: "aa:bb:cc:dd:ee:11"})
	service.Create(CreateNodeRequest{Name: "node-3", IPAddress: "192.168.1.103", MACAddress: "aa:bb:cc:dd:ee:12"})

	t.Run("list all nodes", func(t *testing.T) {
		nodes, count, err := service.List(NodeListOptions{})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))
		assert.GreaterOrEqual(t, len(nodes), 3)
	})
}

func TestNodeService_Update(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create a node first
	created, err := service.Create(CreateNodeRequest{
		Name:       "original-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:30",
	})
	require.NoError(t, err)

	t.Run("update name", func(t *testing.T) {
		newName := "updated-node-name"
		updated, err := service.Update(created.ID, UpdateNodeRequest{
			Name: &newName,
		})
		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newName, updated.Name)
	})

	t.Run("update IP address", func(t *testing.T) {
		newIP := "192.168.1.200"
		updated, err := service.Update(created.ID, UpdateNodeRequest{
			IPAddress: &newIP,
		})
		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newIP, updated.IPAddress)
	})

	t.Run("update non-existent node", func(t *testing.T) {
		newName := "new-name"
		updated, err := service.Update(99999, UpdateNodeRequest{
			Name: &newName,
		})
		assert.Error(t, err)
		assert.Nil(t, updated)
	})
}

func TestNodeService_Delete(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create a node first
	created, err := service.Create(CreateNodeRequest{
		Name:       "node-to-delete",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:40",
	})
	require.NoError(t, err)

	t.Run("delete existing node", func(t *testing.T) {
		err := service.Delete(created.ID)
		assert.NoError(t, err)

		// Verify node was deleted
		node, err := service.GetByID(created.ID, false)
		assert.Error(t, err)
		assert.Nil(t, node)
	})

	t.Run("delete non-existent node", func(t *testing.T) {
		err := service.Delete(99999)
		assert.Error(t, err) // Should error on non-existent node
	})
}

func TestNodeService_UpdateLastSeen(t *testing.T) {
	service, db, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create a node first
	created, err := service.Create(CreateNodeRequest{
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:50",
	})
	require.NoError(t, err)

	t.Run("update last seen timestamp", func(t *testing.T) {
		// Get initial timestamp
		node, err := service.GetByID(created.ID, false)
		require.NoError(t, err)
		initialTime := node.LastSeen

		// Update last seen
		err = service.UpdateLastSeen(created.ID)
		assert.NoError(t, err)

		// Verify timestamp was updated
		var updated models.Node
		db.DB().First(&updated, created.ID)
		assert.True(t, updated.LastSeen.After(initialTime) || updated.LastSeen.Equal(initialTime))
	})

	t.Run("update last seen for non-existent node", func(t *testing.T) {
		err := service.UpdateLastSeen(99999)
		assert.NoError(t, err) // Update doesn't error on non-existent
	})
}

func TestNodeService_GetGPIODevices(t *testing.T) {
	service, db, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create a node first
	node, err := service.Create(CreateNodeRequest{
		Name:       "gpio-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "00:00:00:00:00:01",
	})
	require.NoError(t, err)

	// Create GPIO devices for this node
	gpio1 := &models.GPIODevice{
		NodeID:    node.ID,
		Name:      "GPIO1",
		PinNumber: 1,
	}
	gpio2 := &models.GPIODevice{
		NodeID:    node.ID,
		Name:      "GPIO2",
		PinNumber: 2,
	}

	err = db.DB().Create(gpio1).Error
	require.NoError(t, err)
	err = db.DB().Create(gpio2).Error
	require.NoError(t, err)

	t.Run("get GPIO devices for node with devices", func(t *testing.T) {
		devices, err := service.GetGPIODevices(node.ID)
		assert.NoError(t, err)
		assert.Len(t, devices, 2)
	})

	t.Run("get GPIO devices for node without devices", func(t *testing.T) {
		emptyNode, err := service.Create(CreateNodeRequest{
			Name:       "empty-node",
			IPAddress:  "192.168.1.101",
			MACAddress: "00:00:00:00:00:02",
		})
		require.NoError(t, err)

		devices, err := service.GetGPIODevices(emptyNode.ID)
		assert.NoError(t, err)
		assert.Empty(t, devices)
	})
}

// Discovery and Type Filtering Tests

func TestNodeService_CreateWithDiscoveryInfo(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		req     CreateNodeRequest
		wantErr bool
	}{
		{
			name: "controller discovered via mDNS",
			req: CreateNodeRequest{
				Name:              "pi-controller-1",
				IPAddress:         "192.168.1.10",
				MACAddress:        "aa:bb:cc:dd:ee:70",
				Role:              models.NodeRoleMaster,
				DiscoveryMethod:   models.DiscoveryMethodMDNS,
				NodeType:          models.NodeTypeController,
				ControllerVersion: "v1.0.0",
			},
			wantErr: false,
		},
		{
			name: "agent discovered via mDNS",
			req: CreateNodeRequest{
				Name:            "pi-agent-1",
				IPAddress:       "192.168.1.15",
				MACAddress:      "aa:bb:cc:dd:ee:71",
				Role:            models.NodeRoleWorker,
				DiscoveryMethod: models.DiscoveryMethodMDNS,
				NodeType:        models.NodeTypeAgent,
				AgentPort:       9091,
			},
			wantErr: false,
		},
		{
			name: "generic node manual entry",
			req: CreateNodeRequest{
				Name:            "remote-pi",
				IPAddress:       "10.0.5.50",
				MACAddress:      "aa:bb:cc:dd:ee:72",
				Role:            models.NodeRoleWorker,
				DiscoveryMethod: models.DiscoveryMethodManual,
				NodeType:        models.NodeTypeGeneric,
			},
			wantErr: false,
		},
		{
			name: "controller via Raft cluster",
			req: CreateNodeRequest{
				Name:              "raft-controller",
				IPAddress:         "192.168.1.12",
				MACAddress:        "aa:bb:cc:dd:ee:73",
				Role:              models.NodeRoleMaster,
				DiscoveryMethod:   models.DiscoveryMethodRaftCluster,
				NodeType:          models.NodeTypeController,
				ControllerVersion: "v1.0.0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := service.Create(tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
				assert.Equal(t, tt.req.DiscoveryMethod, node.DiscoveryMethod)
				assert.Equal(t, tt.req.NodeType, node.NodeType)
				assert.Equal(t, tt.req.ControllerVersion, node.ControllerVersion)
				assert.Equal(t, tt.req.AgentPort, node.AgentPort)
			}
		})
	}
}

func TestNodeService_ListByDiscoveryMethod(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create nodes with different discovery methods
	service.Create(CreateNodeRequest{
		Name:            "mdns-node-1",
		IPAddress:       "192.168.1.10",
		MACAddress:      "aa:bb:cc:dd:ee:80",
		DiscoveryMethod: models.DiscoveryMethodMDNS,
		NodeType:        models.NodeTypeController,
	})
	service.Create(CreateNodeRequest{
		Name:            "mdns-node-2",
		IPAddress:       "192.168.1.11",
		MACAddress:      "aa:bb:cc:dd:ee:81",
		DiscoveryMethod: models.DiscoveryMethodMDNS,
		NodeType:        models.NodeTypeAgent,
	})
	service.Create(CreateNodeRequest{
		Name:            "manual-node",
		IPAddress:       "192.168.1.20",
		MACAddress:      "aa:bb:cc:dd:ee:82",
		DiscoveryMethod: models.DiscoveryMethodManual,
		NodeType:        models.NodeTypeGeneric,
	})

	t.Run("filter by mDNS discovery", func(t *testing.T) {
		method := models.DiscoveryMethodMDNS
		nodes, count, err := service.List(NodeListOptions{
			DiscoveryMethod: &method,
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2))
		for _, node := range nodes {
			assert.Equal(t, models.DiscoveryMethodMDNS, node.DiscoveryMethod)
		}
	})

	t.Run("filter by manual discovery", func(t *testing.T) {
		method := models.DiscoveryMethodManual
		nodes, count, err := service.List(NodeListOptions{
			DiscoveryMethod: &method,
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
		for _, node := range nodes {
			assert.Equal(t, models.DiscoveryMethodManual, node.DiscoveryMethod)
		}
	})
}

func TestNodeService_ListByNodeType(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create nodes with different types
	service.Create(CreateNodeRequest{
		Name:            "controller-1",
		IPAddress:       "192.168.1.10",
		MACAddress:      "aa:bb:cc:dd:ee:90",
		DiscoveryMethod: models.DiscoveryMethodMDNS,
		NodeType:        models.NodeTypeController,
	})
	service.Create(CreateNodeRequest{
		Name:            "controller-2",
		IPAddress:       "192.168.1.11",
		MACAddress:      "aa:bb:cc:dd:ee:91",
		DiscoveryMethod: models.DiscoveryMethodMDNS,
		NodeType:        models.NodeTypeController,
	})
	service.Create(CreateNodeRequest{
		Name:            "agent-1",
		IPAddress:       "192.168.1.15",
		MACAddress:      "aa:bb:cc:dd:ee:92",
		DiscoveryMethod: models.DiscoveryMethodMDNS,
		NodeType:        models.NodeTypeAgent,
	})
	service.Create(CreateNodeRequest{
		Name:            "generic-1",
		IPAddress:       "192.168.1.20",
		MACAddress:      "aa:bb:cc:dd:ee:93",
		DiscoveryMethod: models.DiscoveryMethodManual,
		NodeType:        models.NodeTypeGeneric,
	})

	t.Run("filter by controller type", func(t *testing.T) {
		nodeType := models.NodeTypeController
		nodes, count, err := service.List(NodeListOptions{
			NodeType: &nodeType,
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2))
		for _, node := range nodes {
			assert.Equal(t, models.NodeTypeController, node.NodeType)
		}
	})

	t.Run("filter by agent type", func(t *testing.T) {
		nodeType := models.NodeTypeAgent
		nodes, count, err := service.List(NodeListOptions{
			NodeType: &nodeType,
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
		for _, node := range nodes {
			assert.Equal(t, models.NodeTypeAgent, node.NodeType)
		}
	})

	t.Run("filter by generic type", func(t *testing.T) {
		nodeType := models.NodeTypeGeneric
		nodes, count, err := service.List(NodeListOptions{
			NodeType: &nodeType,
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
		for _, node := range nodes {
			assert.Equal(t, models.NodeTypeGeneric, node.NodeType)
		}
	})
}

func TestNodeService_ListByDiscoveryMethodAndType(t *testing.T) {
	service, _, cleanup := setupNodeTest(t)
	defer cleanup()

	// Create various combinations
	service.Create(CreateNodeRequest{
		Name:            "mdns-controller",
		IPAddress:       "192.168.1.10",
		MACAddress:      "aa:bb:cc:dd:ee:a0",
		DiscoveryMethod: models.DiscoveryMethodMDNS,
		NodeType:        models.NodeTypeController,
	})
	service.Create(CreateNodeRequest{
		Name:            "mdns-agent",
		IPAddress:       "192.168.1.15",
		MACAddress:      "aa:bb:cc:dd:ee:a1",
		DiscoveryMethod: models.DiscoveryMethodMDNS,
		NodeType:        models.NodeTypeAgent,
	})
	service.Create(CreateNodeRequest{
		Name:            "manual-generic",
		IPAddress:       "192.168.1.20",
		MACAddress:      "aa:bb:cc:dd:ee:a2",
		DiscoveryMethod: models.DiscoveryMethodManual,
		NodeType:        models.NodeTypeGeneric,
	})

	t.Run("filter by mDNS controller", func(t *testing.T) {
		method := models.DiscoveryMethodMDNS
		nodeType := models.NodeTypeController
		nodes, count, err := service.List(NodeListOptions{
			DiscoveryMethod: &method,
			NodeType:        &nodeType,
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
		for _, node := range nodes {
			assert.Equal(t, models.DiscoveryMethodMDNS, node.DiscoveryMethod)
			assert.Equal(t, models.NodeTypeController, node.NodeType)
		}
	})

	t.Run("filter by manual generic", func(t *testing.T) {
		method := models.DiscoveryMethodManual
		nodeType := models.NodeTypeGeneric
		nodes, count, err := service.List(NodeListOptions{
			DiscoveryMethod: &method,
			NodeType:        &nodeType,
		})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
		for _, node := range nodes {
			assert.Equal(t, models.DiscoveryMethodManual, node.DiscoveryMethod)
			assert.Equal(t, models.NodeTypeGeneric, node.NodeType)
		}
	})
}
