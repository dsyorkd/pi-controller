package services

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/dsyorkd/pi-controller/internal/models"
)

// CAService defines the interface for Certificate Authority operations
type CAService interface {
	// CA Management
	InitializeCA(ctx context.Context) error
	GetCAInfo(ctx context.Context) (*models.CAInfo, error)
	GetCACertificate(ctx context.Context) (*x509.Certificate, error)

	// Certificate Issuance
	IssueCertificate(ctx context.Context, req *IssueCertificateRequest) (*models.Certificate, error)
	RenewCertificate(ctx context.Context, certID uint) (*models.Certificate, error)
	RevokeCertificate(ctx context.Context, certID uint, reason string) error

	// Certificate Management
	GetCertificate(ctx context.Context, id uint) (*models.Certificate, error)
	GetCertificateBySerial(ctx context.Context, serial string) (*models.Certificate, error)
	ListCertificates(ctx context.Context, opts ListCertificatesOptions) ([]*models.Certificate, int64, error)
	ValidateCertificate(ctx context.Context, certPEM string) (*CertificateValidation, error)

	// Certificate Requests (CSR)
	CreateCertificateRequest(ctx context.Context, req *CreateCSRRequest) (*models.CertificateRequest, error)
	ProcessCertificateRequest(ctx context.Context, csrID uint, approve bool) (*models.Certificate, error)
	ListCertificateRequests(ctx context.Context, opts ListCSROptions) ([]*models.CertificateRequest, int64, error)

	// Maintenance
	CleanupExpiredCertificates(ctx context.Context) error
	GetCertificateStats(ctx context.Context) (*CertificateStats, error)
}

// IssueCertificateRequest represents a certificate issuance request
type IssueCertificateRequest struct {
	CommonName     string                 `json:"common_name"`
	Type           models.CertificateType `json:"type"`
	SANs           []string               `json:"sans,omitempty"`
	ValidityPeriod time.Duration          `json:"validity_period,omitempty"`
	KeyUsage       []string               `json:"key_usage,omitempty"`
	ExtKeyUsage    []string               `json:"ext_key_usage,omitempty"`
	NodeID         *uint                  `json:"node_id,omitempty"`
	ClusterID      *uint                  `json:"cluster_id,omitempty"`
	AutoRenew      bool                   `json:"auto_renew"`
}

// CreateCSRRequest represents a certificate signing request creation
type CreateCSRRequest struct {
	CommonName     string                 `json:"common_name"`
	Type           models.CertificateType `json:"type"`
	CSRPEM         string                 `json:"csr_pem"`
	SANs           []string               `json:"sans,omitempty"`
	ValidityPeriod string                 `json:"validity_period,omitempty"`
	KeyUsage       []string               `json:"key_usage,omitempty"`
	ExtKeyUsage    []string               `json:"ext_key_usage,omitempty"`
	NodeID         *uint                  `json:"node_id,omitempty"`
	ClusterID      *uint                  `json:"cluster_id,omitempty"`
}

// ListCertificatesOptions defines options for listing certificates
type ListCertificatesOptions struct {
	Type      *models.CertificateType    `json:"type,omitempty"`
	Status    *models.CertificateStatus  `json:"status,omitempty"`
	NodeID    *uint                      `json:"node_id,omitempty"`
	ClusterID *uint                      `json:"cluster_id,omitempty"`
	Backend   *models.CertificateBackend `json:"backend,omitempty"`
	Limit     int                        `json:"limit,omitempty"`
	Offset    int                        `json:"offset,omitempty"`
}

// ListCSROptions defines options for listing certificate requests
type ListCSROptions struct {
	Status    *models.CSRStatus       `json:"status,omitempty"`
	Type      *models.CertificateType `json:"type,omitempty"`
	NodeID    *uint                   `json:"node_id,omitempty"`
	ClusterID *uint                   `json:"cluster_id,omitempty"`
	Limit     int                     `json:"limit,omitempty"`
	Offset    int                     `json:"offset,omitempty"`
}

// CertificateValidation represents certificate validation results
type CertificateValidation struct {
	Valid        bool      `json:"valid"`
	Expired      bool      `json:"expired"`
	Revoked      bool      `json:"revoked"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	SerialNumber string    `json:"serial_number"`
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	Errors       []string  `json:"errors,omitempty"`
}

// CertificateStats represents CA statistics
type CertificateStats struct {
	TotalCertificates       int64          `json:"total_certificates"`
	ActiveCertificates      int64          `json:"active_certificates"`
	ExpiredCertificates     int64          `json:"expired_certificates"`
	RevokedCertificates     int64          `json:"revoked_certificates"`
	CertificatesIssued24h   int64          `json:"certificates_issued_24h"`
	CertificatesExpiring30d int64          `json:"certificates_expiring_30d"`
	CAInfo                  *models.CAInfo `json:"ca_info"`
}

// CABackend defines the interface that CA backends must implement
type CABackend interface {
	// Backend identification
	Type() models.CertificateBackend

	// CA initialization and management
	InitializeCA(ctx context.Context) error
	GetCAInfo(ctx context.Context) (*models.CAInfo, error)
	GetCACertificate(ctx context.Context) (*x509.Certificate, error)

	// Certificate operations
	IssueCertificate(ctx context.Context, req *IssueCertificateRequest) (certPEM string, err error)
	RevokeCertificate(ctx context.Context, cert *models.Certificate) error
	ValidateCertificate(ctx context.Context, certPEM string) (*CertificateValidation, error)

	// Backend health
	HealthCheck(ctx context.Context) error
}

// SSHExecutor defines the interface for executing commands on remote nodes
type SSHExecutor interface {
	Execute(ctx context.Context, nodeIP string, command string) (output string, err error)
	CopyFile(ctx context.Context, nodeIP string, localPath string, remotePath string) error
	CopyContent(ctx context.Context, nodeIP string, content string, remotePath string) error
}
