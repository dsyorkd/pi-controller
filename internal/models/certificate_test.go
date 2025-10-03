package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCertificate_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		notAfter time.Time
		want     bool
	}{
		{
			name:     "expired certificate",
			notAfter: time.Now().Add(-24 * time.Hour),
			want:     true,
		},
		{
			name:     "valid certificate",
			notAfter: time.Now().Add(24 * time.Hour),
			want:     false,
		},
		{
			name:     "certificate expiring in 1 second",
			notAfter: time.Now().Add(1 * time.Second),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := &Certificate{NotAfter: tt.notAfter}
			assert.Equal(t, tt.want, cert.IsExpired())
		})
	}
}

func TestCertificate_NeedsRenewal(t *testing.T) {
	tests := []struct {
		name      string
		notAfter  time.Time
		autoRenew bool
		status    CertificateStatus
		threshold time.Duration
		want      bool
	}{
		{
			name:      "needs renewal in 1 day",
			notAfter:  time.Now().Add(24 * time.Hour),
			autoRenew: true,
			status:    CertificateStatusActive,
			threshold: 48 * time.Hour,
			want:      true,
		},
		{
			name:      "no renewal needed",
			notAfter:  time.Now().Add(90 * 24 * time.Hour),
			autoRenew: true,
			status:    CertificateStatusActive,
			threshold: 30 * 24 * time.Hour,
			want:      false,
		},
		{
			name:      "auto-renew disabled",
			notAfter:  time.Now().Add(24 * time.Hour),
			autoRenew: false,
			status:    CertificateStatusActive,
			threshold: 48 * time.Hour,
			want:      false,
		},
		{
			name:      "not active",
			notAfter:  time.Now().Add(24 * time.Hour),
			autoRenew: true,
			status:    CertificateStatusRevoked,
			threshold: 48 * time.Hour,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := &Certificate{
				NotAfter:  tt.notAfter,
				AutoRenew: tt.autoRenew,
				Status:    tt.status,
			}
			assert.Equal(t, tt.want, cert.NeedsRenewal(tt.threshold))
		})
	}
}

func TestCertificate_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status CertificateStatus
		want   bool
	}{
		{
			name:   "active certificate",
			status: CertificateStatusActive,
			want:   true,
		},
		{
			name:   "expired certificate",
			status: CertificateStatusExpired,
			want:   false,
		},
		{
			name:   "revoked certificate",
			status: CertificateStatusRevoked,
			want:   false,
		},
		{
			name:   "pending certificate",
			status: CertificateStatusPending,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := &Certificate{Status: tt.status}
			assert.Equal(t, tt.want, cert.IsActive())
		})
	}
}

func TestCertificateType_Constants(t *testing.T) {
	assert.Equal(t, CertificateType("ca"), CertificateTypeCA)
	assert.Equal(t, CertificateType("server"), CertificateTypeServer)
	assert.Equal(t, CertificateType("client"), CertificateTypeClient)
	assert.Equal(t, CertificateType("ssh"), CertificateTypeSSH)
	assert.Equal(t, CertificateType("intermediate"), CertificateTypeIntermediate)
}

func TestCertificateStatus_Constants(t *testing.T) {
	assert.Equal(t, CertificateStatus("active"), CertificateStatusActive)
	assert.Equal(t, CertificateStatus("expired"), CertificateStatusExpired)
	assert.Equal(t, CertificateStatus("revoked"), CertificateStatusRevoked)
	assert.Equal(t, CertificateStatus("pending"), CertificateStatusPending)
	assert.Equal(t, CertificateStatus("failed"), CertificateStatusFailed)
	assert.Equal(t, CertificateStatus("renewing"), CertificateStatusRenewing)
}

func TestCertificateBackend_Constants(t *testing.T) {
	assert.Equal(t, CertificateBackend("local"), CertificateBackendLocal)
	assert.Equal(t, CertificateBackend("vault"), CertificateBackendVault)
}

func TestCSRStatus_Constants(t *testing.T) {
	assert.Equal(t, CSRStatus("pending"), CSRStatusPending)
	assert.Equal(t, CSRStatus("approved"), CSRStatusApproved)
	assert.Equal(t, CSRStatus("rejected"), CSRStatusRejected)
	assert.Equal(t, CSRStatus("failed"), CSRStatusFailed)
}

func TestCertificate_BasicFields(t *testing.T) {
	now := time.Now()
	notAfter := now.Add(365 * 24 * time.Hour)

	cert := &Certificate{
		ID:           1,
		CommonName:   "example.com",
		SerialNumber: "1234567890",
		Type:         CertificateTypeServer,
		Status:       CertificateStatusActive,
		NotBefore:    now,
		NotAfter:     notAfter,
		Backend:      CertificateBackendLocal,
	}

	assert.Equal(t, uint(1), cert.ID)
	assert.Equal(t, "example.com", cert.CommonName)
	assert.Equal(t, "1234567890", cert.SerialNumber)
	assert.Equal(t, CertificateTypeServer, cert.Type)
	assert.Equal(t, CertificateStatusActive, cert.Status)
	assert.Equal(t, CertificateBackendLocal, cert.Backend)
}

func TestCertificate_SANs(t *testing.T) {
	cert := &Certificate{
		CommonName: "example.com",
		SANs:       `["www.example.com", "api.example.com", "192.168.1.1"]`,
	}

	assert.NotEmpty(t, cert.SANs)
	assert.Contains(t, cert.SANs, "www.example.com")
	assert.Contains(t, cert.SANs, "api.example.com")
	assert.Contains(t, cert.SANs, "192.168.1.1")
}

func TestCertificate_Relationships(t *testing.T) {
	t.Run("certificate with node", func(t *testing.T) {
		nodeID := uint(5)
		cert := &Certificate{
			CommonName: "node-cert",
			NodeID:     &nodeID,
		}

		assert.NotNil(t, cert.NodeID)
		assert.Equal(t, uint(5), *cert.NodeID)
	})

	t.Run("certificate with cluster", func(t *testing.T) {
		clusterID := uint(10)
		cert := &Certificate{
			CommonName: "cluster-cert",
			ClusterID:  &clusterID,
		}

		assert.NotNil(t, cert.ClusterID)
		assert.Equal(t, uint(10), *cert.ClusterID)
	})
}

func TestCertificate_RenewalConfig(t *testing.T) {
	cert := &Certificate{
		CommonName: "renewable-cert",
		AutoRenew:  true,
	}

	assert.True(t, cert.AutoRenew)
}

func TestCertificateRequest_BasicFields(t *testing.T) {
	csr := &CertificateRequest{
		ID:         1,
		CommonName: "pending-cert",
		Type:       CertificateTypeClient,
		Status:     CSRStatusPending,
	}

	assert.Equal(t, uint(1), csr.ID)
	assert.Equal(t, "pending-cert", csr.CommonName)
	assert.Equal(t, CertificateTypeClient, csr.Type)
	assert.Equal(t, CSRStatusPending, csr.Status)
}

func TestCertificate_IsRevoked(t *testing.T) {
	tests := []struct {
		name   string
		status CertificateStatus
		want   bool
	}{
		{
			name:   "revoked certificate",
			status: CertificateStatusRevoked,
			want:   true,
		},
		{
			name:   "active certificate",
			status: CertificateStatusActive,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := &Certificate{Status: tt.status}
			assert.Equal(t, tt.want, cert.IsRevoked())
		})
	}
}
