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
			wantErr: false, // Service allows empty names
		},
		{
			name: "missing ip address",
			req: CreateNodeRequest{
				Name:       "test-node-2",
				IPAddress:  "",
				MACAddress: "aa:bb:cc:dd:ee:03",
			},
			wantErr: false, // Service allows empty IP addresses
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
		Name:      "gpio-node",
		IPAddress: "192.168.1.100",
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
