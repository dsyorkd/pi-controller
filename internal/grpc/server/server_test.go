package server

import (
	"context"
	"net"
	"testing"

	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/storage"
	pb "github.com/dsyorkd/pi-controller/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func setupGRPCTest(t *testing.T) (*Server, *grpc.ClientConn, pb.PiControllerServiceClient, *storage.Database, func()) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)

	cfg := &config.GRPCConfig{
		Host: "localhost",
		Port: 0, // Dynamic port
	}

	server, err := New(cfg, log, db)
	require.NoError(t, err)

	// Create in-memory listener for testing
	lis := bufconn.Listen(bufSize)

	// Start server in goroutine
	go func() {
		if err := server.server.Serve(lis); err != nil {
			log.WithError(err).Error("Server failed")
		}
	}()

	// Create client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := pb.NewPiControllerServiceClient(conn)

	cleanup := func() {
		conn.Close()
		server.Stop()
		lis.Close()
		db.Close()
	}

	return server, conn, client, db, cleanup
}

func TestNew(t *testing.T) {
	log := logger.Default()
	db, err := storage.NewForTest(log)
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.GRPCConfig{
		Host: "localhost",
		Port: 50051,
	}

	t.Run("create server successfully", func(t *testing.T) {
		server, err := New(cfg, log, db)
		assert.NoError(t, err)
		assert.NotNil(t, server)
		assert.NotNil(t, server.server)
		assert.NotNil(t, server.config)
		assert.NotNil(t, server.logger)
		assert.NotNil(t, server.database)
	})
}

func TestServer_Health(t *testing.T) {
	_, _, client, _, cleanup := setupGRPCTest(t)
	defer cleanup()

	resp, err := client.Health(context.Background(), &pb.HealthRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)
	assert.NotNil(t, resp.Timestamp)
	assert.Equal(t, "dev", resp.Version)
}

func TestServer_CreateCluster(t *testing.T) {
	_, _, client, _, cleanup := setupGRPCTest(t)
	defer cleanup()

	t.Run("create cluster successfully", func(t *testing.T) {
		req := &pb.CreateClusterRequest{
			Name:           "test-cluster",
			Description:    "Test cluster for gRPC",
			Version:        "v1.28.0",
			MasterEndpoint: "https://192.168.1.100:6443",
		}

		resp, err := client.CreateCluster(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotZero(t, resp.Id)
		assert.Equal(t, req.Name, resp.Name)
		assert.Equal(t, req.Description, resp.Description)
		assert.Equal(t, req.Version, resp.Version)
	})
}

func TestServer_GetCluster(t *testing.T) {
	_, _, client, _, cleanup := setupGRPCTest(t)
	defer cleanup()

	// Create a cluster first
	createReq := &pb.CreateClusterRequest{
		Name:        "test-cluster",
		Description: "Test cluster",
	}
	created, err := client.CreateCluster(context.Background(), createReq)
	require.NoError(t, err)

	t.Run("get existing cluster", func(t *testing.T) {
		resp, err := client.GetCluster(context.Background(), &pb.GetClusterRequest{
			Id: created.Id,
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, created.Id, resp.Id)
		assert.Equal(t, created.Name, resp.Name)
	})

	t.Run("get non-existent cluster", func(t *testing.T) {
		_, err := client.GetCluster(context.Background(), &pb.GetClusterRequest{
			Id: 99999,
		})
		assert.Error(t, err)
	})
}

func TestServer_ListClusters(t *testing.T) {
	_, _, client, _, cleanup := setupGRPCTest(t)
	defer cleanup()

	// Create multiple clusters
	for i := 1; i <= 3; i++ {
		_, err := client.CreateCluster(context.Background(), &pb.CreateClusterRequest{
			Name: "test-cluster-" + string(rune('0'+i)),
		})
		require.NoError(t, err)
	}

	t.Run("list all clusters without pagination", func(t *testing.T) {
		resp, err := client.ListClusters(context.Background(), &pb.ListClustersRequest{})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.GreaterOrEqual(t, len(resp.Clusters), 3)
		assert.GreaterOrEqual(t, int(resp.TotalCount), 3)
	})

	t.Run("list clusters with pagination", func(t *testing.T) {
		resp, err := client.ListClusters(context.Background(), &pb.ListClustersRequest{
			PageSize: 2,
			Page:     1,
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.LessOrEqual(t, len(resp.Clusters), 2)
	})
}

func TestServer_CreateNode(t *testing.T) {
	_, _, client, _, cleanup := setupGRPCTest(t)
	defer cleanup()

	t.Run("create node successfully", func(t *testing.T) {
		req := &pb.CreateNodeRequest{
			Name:         "test-node",
			IpAddress:    "192.168.1.100",
			MacAddress:   "aa:bb:cc:dd:ee:ff",
			Architecture: "arm64",
			Model:        "Raspberry Pi 4",
			SerialNumber: "123456789",
			CpuCores:     4,
			Memory:       8589934592,
		}

		resp, err := client.CreateNode(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotZero(t, resp.Id)
		assert.Equal(t, req.Name, resp.Name)
		assert.Equal(t, req.IpAddress, resp.IpAddress)
		assert.Equal(t, req.MacAddress, resp.MacAddress)
	})

	t.Run("create node with cluster association", func(t *testing.T) {
		// Create cluster first
		cluster, err := client.CreateCluster(context.Background(), &pb.CreateClusterRequest{
			Name: "cluster-for-node",
		})
		require.NoError(t, err)

		clusterID := cluster.Id
		req := &pb.CreateNodeRequest{
			Name:       "node-in-cluster",
			IpAddress:  "192.168.1.101",
			MacAddress: "aa:bb:cc:dd:ee:f0",
			ClusterId:  &clusterID,
		}

		resp, err := client.CreateNode(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotZero(t, resp.Id)
	})
}

func TestServer_GetNode(t *testing.T) {
	_, _, client, _, cleanup := setupGRPCTest(t)
	defer cleanup()

	// Create a node first
	createReq := &pb.CreateNodeRequest{
		Name:       "test-node",
		IpAddress:  "192.168.1.100",
		MacAddress: "aa:bb:cc:dd:ee:ff",
	}
	created, err := client.CreateNode(context.Background(), createReq)
	require.NoError(t, err)

	t.Run("get existing node", func(t *testing.T) {
		resp, err := client.GetNode(context.Background(), &pb.GetNodeRequest{
			Id: created.Id,
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, created.Id, resp.Id)
		assert.Equal(t, created.Name, resp.Name)
	})

	t.Run("get non-existent node", func(t *testing.T) {
		_, err := client.GetNode(context.Background(), &pb.GetNodeRequest{
			Id: 99999,
		})
		assert.Error(t, err)
	})
}

func TestLoggingInterceptor(t *testing.T) {
	log := logger.Default()
	interceptor := loggingInterceptor(log)

	assert.NotNil(t, interceptor)

	// Test that the interceptor can be called
	ctx := context.Background()
	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "response", nil
	}

	resp, err := interceptor(ctx, "request", &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.True(t, called)
}

func TestStreamLoggingInterceptor(t *testing.T) {
	log := logger.Default()
	interceptor := streamLoggingInterceptor(log)

	assert.NotNil(t, interceptor)
}
