package services

import (
	"testing"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupClusterTest(t *testing.T) (*ClusterService, *storage.Database, func()) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)

	service := NewClusterService(db, log)

	cleanup := func() {
		db.Close()
	}

	return service, db, cleanup
}

func TestNewClusterService(t *testing.T) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)
	defer db.Close()

	service := NewClusterService(db, log)

	assert.NotNil(t, service)
	assert.NotNil(t, service.store)
	assert.NotNil(t, service.log)
}

func TestClusterService_Create(t *testing.T) {
	service, _, cleanup := setupClusterTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		req     CreateClusterRequest
		wantErr bool
	}{
		{
			name: "valid cluster creation",
			req: CreateClusterRequest{
				Name:        "test-cluster",
				Description: "Test cluster",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: CreateClusterRequest{
				Name:        "",
				Description: "Test cluster",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster, err := service.Create(tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cluster)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cluster)
				assert.NotZero(t, cluster.ID)
				assert.Equal(t, tt.req.Name, cluster.Name)
				assert.Equal(t, tt.req.Description, cluster.Description)
			}
		})
	}
}

func TestClusterService_GetByID(t *testing.T) {
	service, _, cleanup := setupClusterTest(t)
	defer cleanup()

	// Create a cluster first
	created, err := service.Create(CreateClusterRequest{
		Name:        "test-cluster",
		Description: "Test description",
	})
	require.NoError(t, err)

	t.Run("existing cluster", func(t *testing.T) {
		cluster, err := service.GetByID(created.ID)
		assert.NoError(t, err)
		assert.NotNil(t, cluster)
		assert.Equal(t, created.ID, cluster.ID)
		assert.Equal(t, created.Name, cluster.Name)
	})

	t.Run("non-existent cluster", func(t *testing.T) {
		cluster, err := service.GetByID(99999)
		assert.Error(t, err)
		assert.Nil(t, cluster)
	})
}

func TestClusterService_GetByName(t *testing.T) {
	service, _, cleanup := setupClusterTest(t)
	defer cleanup()

	// Create a cluster first
	created, err := service.Create(CreateClusterRequest{
		Name:        "unique-cluster-name",
		Description: "Test description",
	})
	require.NoError(t, err)

	t.Run("existing cluster by name", func(t *testing.T) {
		cluster, err := service.GetByName(created.Name)
		assert.NoError(t, err)
		assert.NotNil(t, cluster)
		assert.Equal(t, created.ID, cluster.ID)
		assert.Equal(t, created.Name, cluster.Name)
	})

	t.Run("non-existent cluster by name", func(t *testing.T) {
		cluster, err := service.GetByName("non-existent-cluster")
		assert.Error(t, err)
		assert.Nil(t, cluster)
	})
}

func TestClusterService_List(t *testing.T) {
	service, _, cleanup := setupClusterTest(t)
	defer cleanup()

	// Create multiple clusters
	service.Create(CreateClusterRequest{Name: "cluster-1", Description: "Test 1"})
	service.Create(CreateClusterRequest{Name: "cluster-2", Description: "Test 2"})
	service.Create(CreateClusterRequest{Name: "cluster-3", Description: "Test 3"})

	t.Run("list all clusters", func(t *testing.T) {
		clusters, count, err := service.List(ClusterListOptions{})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))
		assert.GreaterOrEqual(t, len(clusters), 3)
	})
}

func TestClusterService_Update(t *testing.T) {
	service, _, cleanup := setupClusterTest(t)
	defer cleanup()

	// Create a cluster first
	created, err := service.Create(CreateClusterRequest{
		Name:        "original-cluster",
		Description: "Original description",
	})
	require.NoError(t, err)

	t.Run("update name", func(t *testing.T) {
		newName := "updated-cluster-name"
		updated, err := service.Update(created.ID, UpdateClusterRequest{
			Name: &newName,
		})
		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newName, updated.Name)
	})

	t.Run("update description", func(t *testing.T) {
		newDesc := "Updated description"
		updated, err := service.Update(created.ID, UpdateClusterRequest{
			Description: &newDesc,
		})
		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newDesc, updated.Description)
	})

	t.Run("update non-existent cluster", func(t *testing.T) {
		newName := "new-name"
		updated, err := service.Update(99999, UpdateClusterRequest{
			Name: &newName,
		})
		assert.Error(t, err)
		assert.Nil(t, updated)
	})
}

func TestClusterService_Delete(t *testing.T) {
	service, _, cleanup := setupClusterTest(t)
	defer cleanup()

	// Create a cluster first
	created, err := service.Create(CreateClusterRequest{
		Name:        "cluster-to-delete",
		Description: "Will be deleted",
	})
	require.NoError(t, err)

	t.Run("delete existing cluster", func(t *testing.T) {
		err := service.Delete(created.ID)
		assert.NoError(t, err)

		// Verify cluster was deleted
		cluster, err := service.GetByID(created.ID)
		assert.Error(t, err)
		assert.Nil(t, cluster)
	})

	t.Run("delete non-existent cluster", func(t *testing.T) {
		err := service.Delete(99999)
		assert.NoError(t, err) // Soft delete doesn't error on non-existent
	})
}

func TestClusterService_GetStatus(t *testing.T) {
	service, _, cleanup := setupClusterTest(t)
	defer cleanup()

	// Create a cluster first
	created, err := service.Create(CreateClusterRequest{
		Name:        "status-cluster",
		Description: "For status check",
	})
	require.NoError(t, err)

	t.Run("get status of existing cluster", func(t *testing.T) {
		status, err := service.GetStatus(created.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, status)
	})

	t.Run("get status of non-existent cluster", func(t *testing.T) {
		status, err := service.GetStatus(99999)
		assert.Error(t, err)
		assert.Empty(t, status)
	})
}

func TestClusterService_GetNodes(t *testing.T) {
	service, db, cleanup := setupClusterTest(t)
	defer cleanup()

	// Create a cluster first
	cluster, err := service.Create(CreateClusterRequest{
		Name:        "node-cluster",
		Description: "Cluster with nodes",
	})
	require.NoError(t, err)

	// Create nodes for this cluster
	clusterID := cluster.ID
	node1 := &models.Node{
		Name:       "node-1",
		ClusterID:  &clusterID,
		Status:     models.NodeStatusReady,
		MACAddress: "aa:bb:cc:dd:ee:c1",
		IPAddress:  "192.168.1.201",
	}
	node2 := &models.Node{
		Name:       "node-2",
		ClusterID:  &clusterID,
		Status:     models.NodeStatusReady,
		MACAddress: "aa:bb:cc:dd:ee:c2",
		IPAddress:  "192.168.1.202",
	}

	err = db.DB().Create(node1).Error
	require.NoError(t, err)
	err = db.DB().Create(node2).Error
	require.NoError(t, err)

	t.Run("get nodes for cluster with nodes", func(t *testing.T) {
		nodes, err := service.GetNodes(cluster.ID)
		assert.NoError(t, err)
		assert.Len(t, nodes, 2)
	})

	t.Run("get nodes for cluster without nodes", func(t *testing.T) {
		emptyCluster, err := service.Create(CreateClusterRequest{
			Name: "empty-cluster",
		})
		require.NoError(t, err)

		nodes, err := service.GetNodes(emptyCluster.ID)
		assert.NoError(t, err)
		assert.Empty(t, nodes)
	})
}

func TestClusterService_SetDependencies(t *testing.T) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)
	defer db.Close()

	service := NewClusterService(db, log)

	// Create mock services
	provisioningService := &ProvisioningService{}
	nodeService := &NodeService{}

	service.SetDependencies(provisioningService, nodeService)

	assert.NotNil(t, service.provisioningService)
	assert.NotNil(t, service.nodeService)
}
