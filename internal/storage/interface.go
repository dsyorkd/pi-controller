package storage

import "github.com/dsyorkd/pi-controller/internal/models"

// Store is the interface for the storage layer
type Store interface {
	CreateCluster(cluster *models.Cluster) error
	GetCluster(id uint) (*models.Cluster, error)
	GetClusters() ([]models.Cluster, error)
	UpdateCluster(cluster *models.Cluster) error
	DeleteCluster(id uint) error

	GetNodesByClusterID(clusterID uint) ([]models.Node, error)

	CreatePin(pin *models.GPIODevice) error
	GetPin(id uint) (*models.GPIODevice, error)
	GetPins() ([]models.GPIODevice, error)
	UpdatePin(pin *models.GPIODevice) error
	DeletePin(id uint) error
}
