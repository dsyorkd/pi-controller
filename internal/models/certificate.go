package models

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Certificate represents a certificate managed by the CA system
type Certificate struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	SerialNumber string      `json:"serial_number" gorm:"uniqueIndex;not null"`
	CommonName   string      `json:"common_name" gorm:"index;not null"`
	Type         CertificateType `json:"type" gorm:"not null"`
	Status       CertificateStatus `json:"status" gorm:"default:'active'"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Certificate data (PEM encoded)
	CertificatePEM string `json:"-" gorm:"type:text;not null"`
	
	// Certificate metadata
	Subject          string    `json:"subject"`
	Issuer           string    `json:"issuer"`
	NotBefore        time.Time `json:"not_before"`
	NotAfter         time.Time `json:"not_after"`
	KeyUsage         string    `json:"key_usage"`
	ExtKeyUsage      string    `json:"ext_key_usage"`
	SANs             string    `json:"sans"` // JSON array of Subject Alternative Names
	
	// CA backend information
	Backend       CertificateBackend `json:"backend" gorm:"not null"`
	VaultPath     string            `json:"vault_path,omitempty"`     // Vault path if using Vault backend
	LocalPath     string            `json:"local_path,omitempty"`     // Local file path if using local backend
	
	// Node association
	NodeID    *uint `json:"node_id,omitempty" gorm:"index"`
	Node      *Node `json:"node,omitempty" gorm:"foreignKey:NodeID"`
	
	// Cluster association (for cluster-wide certificates)
	ClusterID *uint    `json:"cluster_id,omitempty" gorm:"index"`
	Cluster   *Cluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	
	// Renewal information
	RenewedFromID *uint        `json:"renewed_from_id,omitempty" gorm:"index"`
	RenewedFrom   *Certificate `json:"renewed_from,omitempty" gorm:"foreignKey:RenewedFromID"`
	AutoRenew     bool         `json:"auto_renew" gorm:"default:true"`
	RenewedAt     *time.Time   `json:"renewed_at,omitempty"`
	
	// Revocation information
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	RevokedReason string     `json:"revoked_reason,omitempty"`
}

// CertificateType defines the type of certificate
type CertificateType string

const (
	CertificateTypeCA         CertificateType = "ca"          // Root or intermediate CA certificate
	CertificateTypeServer     CertificateType = "server"      // Server/service certificate
	CertificateTypeClient     CertificateType = "client"      // Client authentication certificate
	CertificateTypeSSH        CertificateType = "ssh"         // SSH certificate
	CertificateTypeIntermediate CertificateType = "intermediate" // Intermediate CA certificate
)

// CertificateStatus defines the current status of a certificate
type CertificateStatus string

const (
	CertificateStatusActive    CertificateStatus = "active"    // Certificate is valid and active
	CertificateStatusExpired   CertificateStatus = "expired"   // Certificate has expired
	CertificateStatusRevoked   CertificateStatus = "revoked"   // Certificate has been revoked
	CertificateStatusPending   CertificateStatus = "pending"   // Certificate is being issued
	CertificateStatusFailed    CertificateStatus = "failed"    // Certificate issuance failed
	CertificateStatusRenewing  CertificateStatus = "renewing"  // Certificate is being renewed
)

// CertificateBackend defines which CA backend was used to issue the certificate
type CertificateBackend string

const (
	CertificateBackendLocal CertificateBackend = "local" // Local CA backend
	CertificateBackendVault CertificateBackend = "vault" // Vault PKI backend
)

// IsActive returns true if the certificate is in an active state
func (c *Certificate) IsActive() bool {
	return c.Status == CertificateStatusActive
}

// IsExpired returns true if the certificate has expired
func (c *Certificate) IsExpired() bool {
	return c.Status == CertificateStatusExpired || time.Now().After(c.NotAfter)
}

// IsRevoked returns true if the certificate has been revoked
func (c *Certificate) IsRevoked() bool {
	return c.Status == CertificateStatusRevoked
}

// NeedsRenewal returns true if the certificate should be renewed
func (c *Certificate) NeedsRenewal(threshold time.Duration) bool {
	if !c.IsActive() || !c.AutoRenew {
		return false
	}
	return time.Now().Add(threshold).After(c.NotAfter)
}

// GetX509Certificate parses and returns the X.509 certificate from PEM data
func (c *Certificate) GetX509Certificate() (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(c.CertificatePEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, ErrInvalidCertificatePEM
	}
	
	return x509.ParseCertificate(block.Bytes)
}

// TableName returns the table name for the Certificate model
func (Certificate) TableName() string {
	return "certificates"
}

// CertificateRequest represents a certificate signing request
type CertificateRequest struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CommonName string        `json:"common_name" gorm:"not null"`
	Type      CertificateType `json:"type" gorm:"not null"`
	Status    CSRStatus      `json:"status" gorm:"default:'pending'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// CSR data (PEM encoded)
	CSRPEM string `json:"-" gorm:"type:text;not null"`
	
	// Request parameters
	SANs             string `json:"sans"`              // JSON array of Subject Alternative Names
	ValidityPeriod   string `json:"validity_period"`  // Duration string (e.g., "8760h")
	KeyUsage         string `json:"key_usage"`         // Comma-separated key usage
	ExtKeyUsage      string `json:"ext_key_usage"`     // Comma-separated extended key usage
	
	// Node association
	NodeID    *uint `json:"node_id,omitempty" gorm:"index"`
	Node      *Node `json:"node,omitempty" gorm:"foreignKey:NodeID"`
	
	// Cluster association
	ClusterID *uint    `json:"cluster_id,omitempty" gorm:"index"`
	Cluster   *Cluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	
	// Processing information
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	CertificateID  *uint      `json:"certificate_id,omitempty" gorm:"index"`
	Certificate    *Certificate `json:"certificate,omitempty" gorm:"foreignKey:CertificateID"`
	FailureReason  string     `json:"failure_reason,omitempty"`
}

// CSRStatus defines the status of a certificate signing request
type CSRStatus string

const (
	CSRStatusPending   CSRStatus = "pending"   // Request is pending processing
	CSRStatusApproved  CSRStatus = "approved"  // Request has been approved and certificate issued
	CSRStatusRejected  CSRStatus = "rejected"  // Request has been rejected
	CSRStatusFailed    CSRStatus = "failed"    // Request processing failed
)

// IsPending returns true if the CSR is pending processing
func (csr *CertificateRequest) IsPending() bool {
	return csr.Status == CSRStatusPending
}

// IsApproved returns true if the CSR has been approved and certificate issued
func (csr *CertificateRequest) IsApproved() bool {
	return csr.Status == CSRStatusApproved
}

// TableName returns the table name for the CertificateRequest model
func (CertificateRequest) TableName() string {
	return "certificate_requests"
}

// CAInfo represents Certificate Authority information
type CAInfo struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Name      string         `json:"name" gorm:"uniqueIndex;not null"`
	Type      CAType         `json:"type" gorm:"not null"`
	Backend   CertificateBackend `json:"backend" gorm:"not null"`
	Status    CAStatus       `json:"status" gorm:"default:'active'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// CA certificate information
	CertificateID *uint        `json:"certificate_id,omitempty" gorm:"index"`
	Certificate   *Certificate `json:"certificate,omitempty" gorm:"foreignKey:CertificateID"`
	
	// Backend-specific configuration
	LocalPath  string `json:"local_path,omitempty"`  // Path to CA files for local backend
	VaultPath  string `json:"vault_path,omitempty"`  // Vault mount path for Vault backend
	
	// CA metadata
	Subject       string    `json:"subject"`
	NotBefore     time.Time `json:"not_before"`
	NotAfter      time.Time `json:"not_after"`
	SerialNumber  string    `json:"serial_number"`
	
	// Statistics
	CertificatesIssued int `json:"certificates_issued" gorm:"default:0"`
	CertificatesActive int `json:"certificates_active" gorm:"default:0"`
}

// CAType defines the type of Certificate Authority
type CAType string

const (
	CATypeRoot         CAType = "root"         // Root CA
	CATypeIntermediate CAType = "intermediate" // Intermediate CA
)

// CAStatus defines the status of a Certificate Authority
type CAStatus string

const (
	CAStatusActive   CAStatus = "active"   // CA is active and can issue certificates
	CAStatusInactive CAStatus = "inactive" // CA is inactive
	CAStatusRevoked  CAStatus = "revoked"  // CA certificate has been revoked
	CAStatusExpired  CAStatus = "expired"  // CA certificate has expired
)

// IsActive returns true if the CA is active and can issue certificates
func (ca *CAInfo) IsActive() bool {
	return ca.Status == CAStatusActive
}

// IsExpired returns true if the CA certificate has expired
func (ca *CAInfo) IsExpired() bool {
	return ca.Status == CAStatusExpired || time.Now().After(ca.NotAfter)
}

// TableName returns the table name for the CAInfo model
func (CAInfo) TableName() string {
	return "ca_info"
}

// Certificate-related errors
var (
	ErrInvalidCertificatePEM = fmt.Errorf("invalid certificate PEM data")
	ErrCertificateNotFound   = fmt.Errorf("certificate not found")
	ErrCertificateExpired    = fmt.Errorf("certificate has expired")
	ErrCertificateRevoked    = fmt.Errorf("certificate has been revoked")
	ErrCSRNotFound          = fmt.Errorf("certificate request not found")
	ErrCANotFound           = fmt.Errorf("certificate authority not found")
	ErrCAInactive           = fmt.Errorf("certificate authority is not active")
)