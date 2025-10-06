package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
	pb "github.com/dsyorkd/pi-controller/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func setupPiControllerTest(t *testing.T) (*PiControllerServer, *storage.Database, *middleware.AuthManager, func()) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)

	authConfig := &middleware.AuthConfig{
		JWTSecret:          []byte("test-secret-key-for-testing-only"),
		AccessTokenExpiry:  3600 * time.Second,
		RefreshTokenExpiry: 86400 * time.Second,
		RequireHTTPS:       false,
	}
	authManager, err := middleware.NewAuthManager(authConfig, log)
	require.NoError(t, err)

	server := NewPiControllerServer(db, log, authManager)

	cleanup := func() {
		db.Close()
	}

	return server, db, authManager, cleanup
}

func TestNewPiControllerServer(t *testing.T) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)
	defer db.Close()

	authConfig := &middleware.AuthConfig{
		JWTSecret:         []byte("test-secret"),
		AccessTokenExpiry: 3600 * time.Second,
	}
	authManager, err := middleware.NewAuthManager(authConfig, log)
	require.NoError(t, err)

	server := NewPiControllerServer(db, log, authManager)

	assert.NotNil(t, server)
	assert.NotNil(t, server.database)
	assert.NotNil(t, server.logger)
	assert.NotNil(t, server.authManager)
}

func TestPiControllerServer_Health(t *testing.T) {
	server, _, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	resp, err := server.Health(context.Background(), &pb.HealthRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)
	assert.NotNil(t, resp.Timestamp)
}

func TestPiControllerServer_CreateCluster(t *testing.T) {
	server, _, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	req := &pb.CreateClusterRequest{
		Name:           "test-cluster",
		Description:    "Test description",
		Version:        "v1.28.0",
		MasterEndpoint: "https://192.168.1.100:6443",
	}

	resp, err := server.CreateCluster(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotZero(t, resp.Id)
	assert.Equal(t, req.Name, resp.Name)
}

func TestPiControllerServer_GetCluster(t *testing.T) {
	server, db, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	// Create test cluster
	cluster := &models.Cluster{
		Name:        "test-cluster",
		Description: "Test",
		Status:      models.ClusterStatusActive,
	}
	err := db.DB().Create(cluster).Error
	require.NoError(t, err)

	t.Run("get existing cluster", func(t *testing.T) {
		resp, err := server.GetCluster(context.Background(), &pb.GetClusterRequest{
			Id: uint32(cluster.ID),
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, cluster.Name, resp.Name)
	})

	t.Run("get non-existent cluster", func(t *testing.T) {
		resp, err := server.GetCluster(context.Background(), &pb.GetClusterRequest{
			Id: 99999,
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestPiControllerServer_CreateNode(t *testing.T) {
	server, _, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	req := &pb.CreateNodeRequest{
		Name:         "test-node",
		IpAddress:    "192.168.1.100",
		MacAddress:   "aa:bb:cc:dd:ee:ff",
		Architecture: "arm64",
		Model:        "Raspberry Pi 4",
		CpuCores:     4,
		Memory:       8589934592,
	}

	resp, err := server.CreateNode(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotZero(t, resp.Id)
}

func TestPiControllerServer_GetNode(t *testing.T) {
	server, db, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	// Create test node
	node := &models.Node{
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:ff",
		Status:     models.NodeStatusReady,
	}
	err := db.DB().Create(node).Error
	require.NoError(t, err)

	t.Run("get existing node", func(t *testing.T) {
		resp, err := server.GetNode(context.Background(), &pb.GetNodeRequest{
			Id: uint32(node.ID),
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, node.Name, resp.Name)
	})

	t.Run("get non-existent node", func(t *testing.T) {
		resp, err := server.GetNode(context.Background(), &pb.GetNodeRequest{
			Id: 99999,
		})
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestPiControllerServer_ReadGPIO_NoAuth(t *testing.T) {
	server, _, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	ctx := context.Background()

	resp, err := server.ReadGPIO(ctx, &pb.ReadGPIORequest{
		Id: 1,
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestPiControllerServer_ReadGPIO_WithAuth(t *testing.T) {
	server, db, authManager, cleanup := setupPiControllerTest(t)
	defer cleanup()

	// Create test user
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         models.RoleViewer,
		IsActive:     true,
	}
	err := db.DB().Create(user).Error
	require.NoError(t, err)

	// Generate token
	token, err := authManager.GenerateToken(fmt.Sprintf("%d", user.ID), string(user.Role), "access")
	require.NoError(t, err)

	// Create test GPIO device
	node := &models.Node{
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:01",
		Status:     models.NodeStatusReady,
	}
	err = db.DB().Create(node).Error
	require.NoError(t, err)

	device := &models.GPIODevice{
		NodeID:    node.ID,
		Name:      "Test GPIO",
		PinNumber: 1,
		Direction: "input",
		Status:    "active",
		Value:     0,
	}
	err = db.DB().Create(device).Error
	require.NoError(t, err)

	// Create context with auth metadata
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := server.ReadGPIO(ctx, &pb.ReadGPIORequest{
		Id: uint32(device.ID),
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint32(device.ID), resp.DeviceId)
	assert.Equal(t, int32(device.PinNumber), resp.Pin)
}

func TestPiControllerServer_WriteGPIO_NoAuth(t *testing.T) {
	server, _, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	ctx := context.Background()

	resp, err := server.WriteGPIO(ctx, &pb.WriteGPIORequest{
		Id:    1,
		Value: 1.0,
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestPiControllerServer_WriteGPIO_InsufficientRole(t *testing.T) {
	server, db, authManager, cleanup := setupPiControllerTest(t)
	defer cleanup()

	// Create test user with viewer role (insufficient for GPIO write)
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         models.RoleViewer,
		IsActive:     true,
	}
	err := db.DB().Create(user).Error
	require.NoError(t, err)

	token, err := authManager.GenerateToken(fmt.Sprintf("%d", user.ID), string(user.Role), "access")
	require.NoError(t, err)

	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := server.WriteGPIO(ctx, &pb.WriteGPIORequest{
		Id:    1,
		Value: 1.0,
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestPiControllerServer_WriteGPIO_Success(t *testing.T) {
	server, db, authManager, cleanup := setupPiControllerTest(t)
	defer cleanup()

	// Create test user with operator role
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         models.RoleOperator,
		IsActive:     true,
	}
	err := db.DB().Create(user).Error
	require.NoError(t, err)

	token, err := authManager.GenerateToken(fmt.Sprintf("%d", user.ID), string(user.Role), "access")
	require.NoError(t, err)

	// Create test GPIO device with output direction
	node := &models.Node{
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:02",
		Status:     models.NodeStatusReady,
	}
	err = db.DB().Create(node).Error
	require.NoError(t, err)

	device := &models.GPIODevice{
		NodeID:    node.ID,
		Name:      "Test Output GPIO",
		PinNumber: 2,
		Direction: "output",
		Status:    "active",
		Value:     0,
	}
	err = db.DB().Create(device).Error
	require.NoError(t, err)

	// Create context with auth metadata
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := server.WriteGPIO(ctx, &pb.WriteGPIORequest{
		Id:    uint32(device.ID),
		Value: 1.0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint32(device.ID), resp.DeviceId)
	assert.Equal(t, int32(1), resp.Value)
}

func TestPiControllerServer_WriteGPIO_NotOutput(t *testing.T) {
	server, db, authManager, cleanup := setupPiControllerTest(t)
	defer cleanup()

	// Create operator user
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         models.RoleOperator,
		IsActive:     true,
	}
	err := db.DB().Create(user).Error
	require.NoError(t, err)

	token, err := authManager.GenerateToken(fmt.Sprintf("%d", user.ID), string(user.Role), "access")
	require.NoError(t, err)

	// Create GPIO device with input direction (not suitable for write)
	node := &models.Node{
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:03",
		Status:     models.NodeStatusReady,
	}
	err = db.DB().Create(node).Error
	require.NoError(t, err)

	device := &models.GPIODevice{
		NodeID:    node.ID,
		Name:      "Test Input GPIO",
		PinNumber: 3,
		Direction: "input",
		Status:    "active",
		Value:     0,
	}
	err = db.DB().Create(device).Error
	require.NoError(t, err)

	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := server.WriteGPIO(ctx, &pb.WriteGPIORequest{
		Id:    uint32(device.ID),
		Value: 1.0,
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestPiControllerServer_ClusterToProto(t *testing.T) {
	server, _, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	cluster := &models.Cluster{
		ID:             1,
		Name:           "test-cluster",
		Description:    "Test",
		Version:        "v1.28.0",
		MasterEndpoint: "https://192.168.1.100:6443",
	}

	pb := server.clusterToProto(cluster)
	assert.NotNil(t, pb)
	assert.Equal(t, uint32(cluster.ID), pb.Id)
	assert.Equal(t, cluster.Name, pb.Name)
	assert.Equal(t, cluster.Description, pb.Description)
}

func TestPiControllerServer_NodeToProto(t *testing.T) {
	server, _, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	node := &models.Node{
		ID:         1,
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:ff",
	}

	pb := server.nodeToProto(node)
	assert.NotNil(t, pb)
	assert.Equal(t, uint32(node.ID), pb.Id)
	assert.Equal(t, node.Name, pb.Name)
	assert.Equal(t, node.IPAddress, pb.IpAddress)
	assert.Equal(t, node.MACAddress, pb.MacAddress)
}

func TestPiControllerServer_ValidateAuthentication(t *testing.T) {
	server, db, authManager, cleanup := setupPiControllerTest(t)
	defer cleanup()

	// Create test user
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         models.RoleViewer,
		IsActive:     true,
	}
	err := db.DB().Create(user).Error
	require.NoError(t, err)

	token, err := authManager.GenerateToken(fmt.Sprintf("%d", user.ID), string(user.Role), "access")
	require.NoError(t, err)

	t.Run("missing metadata", func(t *testing.T) {
		ctx := context.Background()
		_, err := server.validateAuthentication(ctx)
		assert.Error(t, err)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		md := metadata.New(map[string]string{})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		_, err := server.validateAuthentication(ctx)
		assert.Error(t, err)
	})

	t.Run("invalid token format", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"authorization": "InvalidFormat",
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		_, err := server.validateAuthentication(ctx)
		assert.Error(t, err)
	})

	t.Run("valid token", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"authorization": "Bearer " + token,
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		claims, err := server.validateAuthentication(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, fmt.Sprintf("%d", user.ID), claims.UserID)
	})
}

func TestPiControllerServer_RequireRole(t *testing.T) {
	server, _, _, cleanup := setupPiControllerTest(t)
	defer cleanup()

	t.Run("nil claims", func(t *testing.T) {
		err := server.requireRole(nil, middleware.RoleViewer)
		assert.Error(t, err)
	})

	t.Run("admin can access everything", func(t *testing.T) {
		claims := &middleware.JWTClaims{Role: middleware.RoleAdmin}
		assert.NoError(t, server.requireRole(claims, middleware.RoleAdmin))
		assert.NoError(t, server.requireRole(claims, middleware.RoleOperator))
		assert.NoError(t, server.requireRole(claims, middleware.RoleViewer))
	})

	t.Run("operator permissions", func(t *testing.T) {
		claims := &middleware.JWTClaims{Role: middleware.RoleOperator}
		assert.Error(t, server.requireRole(claims, middleware.RoleAdmin))
		assert.NoError(t, server.requireRole(claims, middleware.RoleOperator))
		assert.NoError(t, server.requireRole(claims, middleware.RoleViewer))
	})

	t.Run("viewer permissions", func(t *testing.T) {
		claims := &middleware.JWTClaims{Role: middleware.RoleViewer}
		assert.Error(t, server.requireRole(claims, middleware.RoleAdmin))
		assert.Error(t, server.requireRole(claims, middleware.RoleOperator))
		assert.NoError(t, server.requireRole(claims, middleware.RoleViewer))
	})
}
