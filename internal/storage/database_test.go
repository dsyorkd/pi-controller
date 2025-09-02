package storage

import (
	"os"
	"testing"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("should create a new database connection", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "pi-controller-test")
		assert.NoError(t, err)
		defer os.RemoveAll(dir)

		dbPath := dir + "/test.db"
		logger := logger.Default()
		config := &Config{
			Path: dbPath,
		}

		db, err := New(config, logger)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		_, err = os.Stat(dbPath)
		assert.NoError(t, err)

		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("should create a cluster", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "pi-controller-test")
		assert.NoError(t, err)
		defer os.RemoveAll(dir)

		dbPath := dir + "/test.db"
		logger := logger.Default()
		config := &Config{
			Path: dbPath,
		}

		db, err := New(config, logger)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		cluster := &models.Cluster{
			Name:        "test-cluster",
			Description: "A test cluster",
		}

		err = db.CreateCluster(cluster)
		assert.NoError(t, err)
		assert.NotZero(t, cluster.ID)

		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("should get a cluster", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "pi-controller-test")
		assert.NoError(t, err)
		defer os.RemoveAll(dir)

		dbPath := dir + "/test.db"
		logger := logger.Default()
		config := &Config{
			Path: dbPath,
		}

		db, err := New(config, logger)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		cluster := &models.Cluster{
			Name:        "test-cluster",
			Description: "A test cluster",
		}

		err = db.CreateCluster(cluster)
		assert.NoError(t, err)
		assert.NotZero(t, cluster.ID)

		retrievedCluster, err := db.GetCluster(cluster.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedCluster)
		assert.Equal(t, cluster.Name, retrievedCluster.Name)
		assert.Equal(t, cluster.Description, retrievedCluster.Description)

		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("should get all clusters", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "pi-controller-test")
		assert.NoError(t, err)
		defer os.RemoveAll(dir)

		dbPath := dir + "/test.db"
		logger := logger.Default()
		config := &Config{
			Path: dbPath,
		}

		db, err := New(config, logger)
		assert.NoError(t, err)

		cluster1 := &models.Cluster{Name: "cluster1"}
		cluster2 := &models.Cluster{Name: "cluster2"}
		db.CreateCluster(cluster1)
		db.CreateCluster(cluster2)

		clusters, err := db.GetClusters()
		assert.NoError(t, err)
		assert.Len(t, clusters, 2)

		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("should update a cluster", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "pi-controller-test")
		assert.NoError(t, err)
		defer os.RemoveAll(dir)

		dbPath := dir + "/test.db"
		logger := logger.Default()
		config := &Config{
			Path: dbPath,
		}

		db, err := New(config, logger)
		assert.NoError(t, err)

		cluster := &models.Cluster{Name: "cluster1"}
		db.CreateCluster(cluster)

		cluster.Name = "cluster2"
		err = db.UpdateCluster(cluster)
		assert.NoError(t, err)

		retrievedCluster, err := db.GetCluster(cluster.ID)
		assert.NoError(t, err)
		assert.Equal(t, "cluster2", retrievedCluster.Name)

		err = db.Close()
		assert.NoError(t, err)
	})

	t.Run("should delete a cluster", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "pi-controller-test")
		assert.NoError(t, err)
		defer os.RemoveAll(dir)

		dbPath := dir + "/test.db"
		logger := logger.Default()
		config := &Config{
			Path: dbPath,
		}

		db, err := New(config, logger)
		assert.NoError(t, err)

		cluster := &models.Cluster{Name: "cluster1"}
		db.CreateCluster(cluster)

		err = db.DeleteCluster(cluster.ID)
		assert.NoError(t, err)

		_, err = db.GetCluster(cluster.ID)
		assert.Error(t, err)

		err = db.Close()
		assert.NoError(t, err)
	})
}
