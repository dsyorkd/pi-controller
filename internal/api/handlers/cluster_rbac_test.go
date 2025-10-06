package handlers

import (
	"testing"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestClusterHandler_hasPermission(t *testing.T) {
	handler := &ClusterHandler{
		logger: logger.Default(),
	}

	tests := []struct {
		name         string
		userRole     string
		requiredRole string
		expected     bool
	}{
		{
			name:         "Admin can access operator endpoints",
			userRole:     middleware.RoleAdmin,
			requiredRole: middleware.RoleOperator,
			expected:     true,
		},
		{
			name:         "Operator can access operator endpoints",
			userRole:     middleware.RoleOperator,
			requiredRole: middleware.RoleOperator,
			expected:     true,
		},
		{
			name:         "Viewer cannot access operator endpoints",
			userRole:     middleware.RoleViewer,
			requiredRole: middleware.RoleOperator,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.hasPermission(tt.userRole, tt.requiredRole)
			assert.Equal(t, tt.expected, result)
		})
	}
}