package services

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// CAServiceImpl implements the CAService interface
type CAServiceImpl struct {
	config      *config.CAConfig
	database    *storage.Database
	logger      logger.Interface
	backend     CABackend
	sshExecutor SSHExecutor
}

// NewCAService creates a new CA service instance
func NewCAService(
	config *config.CAConfig,
	database *storage.Database,
	logger logger.Interface,
	sshExecutor SSHExecutor,
) (*CAServiceImpl, error) {
	service := &CAServiceImpl{
		config:      config,
		database:    database,
		logger:      logger.WithField("component", "ca-service"),
		sshExecutor: sshExecutor,
	}
	
	// Initialize the appropriate backend
	if err := service.initializeBackend(); err != nil {
		return nil, fmt.Errorf("failed to initialize CA backend: %w", err)
	}
	
	return service, nil
}

// initializeBackend initializes the CA backend based on configuration
func (s *CAServiceImpl) initializeBackend() error {
	switch s.config.Backend {
	case "local":
		s.backend = NewLocalCABackend(&s.config.Local, &s.config.SSH, s.database, s.logger, s.sshExecutor)
		s.logger.Info("Using Local CA backend")
	case "vault":
		var err error
		s.backend, err = NewVaultCABackend(&s.config.Vault, s.database, s.logger)
		if err != nil {
			return fmt.Errorf("failed to create Vault CA backend: %w", err)
		}
		s.logger.Info("Using Vault CA backend")
	default:
		return fmt.Errorf("unsupported CA backend: %s", s.config.Backend)
	}
	
	return nil
}

// InitializeCA initializes the Certificate Authority
func (s *CAServiceImpl) InitializeCA(ctx context.Context) error {
	s.logger.Info("Initializing Certificate Authority")
	return s.backend.InitializeCA(ctx)
}

// GetCAInfo returns CA information
func (s *CAServiceImpl) GetCAInfo(ctx context.Context) (*models.CAInfo, error) {
	return s.backend.GetCAInfo(ctx)
}

// GetCACertificate returns the CA certificate
func (s *CAServiceImpl) GetCACertificate(ctx context.Context) (*x509.Certificate, error) {
	return s.backend.GetCACertificate(ctx)
}

// IssueCertificate issues a new certificate
func (s *CAServiceImpl) IssueCertificate(ctx context.Context, req *IssueCertificateRequest) (*models.Certificate, error) {
	s.logger.WithFields(map[string]interface{}{
		"common_name": req.CommonName,
		"type":        req.Type,
		"node_id":     req.NodeID,
		"cluster_id":  req.ClusterID,
	}).Info("Issuing certificate")
	
	// Validate request
	if err := s.validateCertificateRequest(req); err != nil {
		return nil, fmt.Errorf("certificate request validation failed: %w", err)
	}
	
	// Issue certificate through backend
	certPEM, err := s.backend.IssueCertificate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("backend certificate issuance failed: %w", err)
	}
	
	// Parse the certificate to extract metadata
	cert, err := s.parseCertificatePEM(certPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse issued certificate: %w", err)
	}
	
	// Store certificate in database
	certRecord := &models.Certificate{
		SerialNumber:   cert.SerialNumber.String(),
		CommonName:     req.CommonName,
		Type:          req.Type,
		Status:        models.CertificateStatusActive,
		CertificatePEM: certPEM,
		Subject:       cert.Subject.String(),
		Issuer:        cert.Issuer.String(),
		NotBefore:     cert.NotBefore,
		NotAfter:      cert.NotAfter,
		Backend:       s.backend.Type(),
		NodeID:        req.NodeID,
		ClusterID:     req.ClusterID,
		AutoRenew:     req.AutoRenew,
	}
	
	// Set key usage and extended key usage
	certRecord.KeyUsage = s.formatKeyUsage(cert.KeyUsage)
	certRecord.ExtKeyUsage = s.formatExtKeyUsage(cert.ExtKeyUsage)
	
	// Set Subject Alternative Names
	if len(cert.DNSNames) > 0 || len(cert.IPAddresses) > 0 || len(cert.EmailAddresses) > 0 {
		sans := make(map[string][]string)
		if len(cert.DNSNames) > 0 {
			sans["dns"] = cert.DNSNames
		}
		if len(cert.IPAddresses) > 0 {
			var ips []string
			for _, ip := range cert.IPAddresses {
				ips = append(ips, ip.String())
			}
			sans["ip"] = ips
		}
		if len(cert.EmailAddresses) > 0 {
			sans["email"] = cert.EmailAddresses
		}
		
		sansJSON, _ := json.Marshal(sans)
		certRecord.SANs = string(sansJSON)
	}
	
	// Set backend-specific paths
	if s.backend.Type() == models.CertificateBackendLocal {
		certRecord.LocalPath = fmt.Sprintf("%s/%s.crt", s.config.Local.DataDir, cert.SerialNumber.String())
	}
	
	if err := s.database.DB().Create(certRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}
	
	// Update CA statistics
	if err := s.updateCAStats(ctx, 1, 0, 0); err != nil {
		s.logger.WithError(err).Warn("Failed to update CA statistics")
	}
	
	s.logger.WithFields(map[string]interface{}{
		"cert_id":       certRecord.ID,
		"serial_number": certRecord.SerialNumber,
		"common_name":   certRecord.CommonName,
		"not_after":     certRecord.NotAfter,
	}).Info("Certificate issued successfully")
	
	return certRecord, nil
}

// RenewCertificate renews an existing certificate
func (s *CAServiceImpl) RenewCertificate(ctx context.Context, certID uint) (*models.Certificate, error) {
	s.logger.WithField("cert_id", certID).Info("Renewing certificate")
	
	// Get the existing certificate
	var existingCert models.Certificate
	if err := s.database.DB().First(&existingCert, certID).Error; err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}
	
	if !existingCert.IsActive() {
		return nil, fmt.Errorf("cannot renew inactive certificate")
	}
	
	// Create renewal request based on existing certificate
	req := &IssueCertificateRequest{
		CommonName:  existingCert.CommonName,
		Type:       existingCert.Type,
		NodeID:     existingCert.NodeID,
		ClusterID:  existingCert.ClusterID,
		AutoRenew:  existingCert.AutoRenew,
	}
	
	// Parse SANs from existing certificate
	if existingCert.SANs != "" {
		var sans map[string][]string
		if err := json.Unmarshal([]byte(existingCert.SANs), &sans); err == nil {
			for _, dnsNames := range sans["dns"] {
				req.SANs = append(req.SANs, dnsNames)
			}
			for _, ips := range sans["ip"] {
				req.SANs = append(req.SANs, ips)
			}
		}
	}
	
	// Use default validity period for renewal
	if s.config.CertificateConfig.DefaultValidityPeriod != "" {
		if duration, err := time.ParseDuration(s.config.CertificateConfig.DefaultValidityPeriod); err == nil {
			req.ValidityPeriod = duration
		}
	}
	
	// Issue the new certificate
	newCert, err := s.IssueCertificate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to issue renewal certificate: %w", err)
	}
	
	// Update the new certificate to reference the old one
	newCert.RenewedFromID = &existingCert.ID
	renewedAt := time.Now()
	newCert.RenewedAt = &renewedAt
	
	if err := s.database.DB().Save(newCert).Error; err != nil {
		s.logger.WithError(err).Warn("Failed to update renewal reference")
	}
	
	// Mark the old certificate as expired
	existingCert.Status = models.CertificateStatusExpired
	if err := s.database.DB().Save(&existingCert).Error; err != nil {
		s.logger.WithError(err).Warn("Failed to mark old certificate as expired")
	}
	
	s.logger.WithFields(map[string]interface{}{
		"old_cert_id": existingCert.ID,
		"new_cert_id": newCert.ID,
		"common_name": newCert.CommonName,
	}).Info("Certificate renewed successfully")
	
	return newCert, nil
}

// RevokeCertificate revokes a certificate
func (s *CAServiceImpl) RevokeCertificate(ctx context.Context, certID uint, reason string) error {
	s.logger.WithFields(map[string]interface{}{
		"cert_id": certID,
		"reason":  reason,
	}).Info("Revoking certificate")
	
	// Get the certificate
	var cert models.Certificate
	if err := s.database.DB().First(&cert, certID).Error; err != nil {
		return fmt.Errorf("certificate not found: %w", err)
	}
	
	if cert.Status == models.CertificateStatusRevoked {
		return fmt.Errorf("certificate is already revoked")
	}
	
	// Revoke through backend
	if err := s.backend.RevokeCertificate(ctx, &cert); err != nil {
		return fmt.Errorf("backend revocation failed: %w", err)
	}
	
	// Update certificate status in database
	cert.Status = models.CertificateStatusRevoked
	revokedAt := time.Now()
	cert.RevokedAt = &revokedAt
	cert.RevokedReason = reason
	
	if err := s.database.DB().Save(&cert).Error; err != nil {
		return fmt.Errorf("failed to update certificate status: %w", err)
	}
	
	// Update CA statistics
	if err := s.updateCAStats(ctx, 0, 0, 1); err != nil {
		s.logger.WithError(err).Warn("Failed to update CA statistics")
	}
	
	s.logger.WithField("cert_id", certID).Info("Certificate revoked successfully")
	return nil
}

// GetCertificate retrieves a certificate by ID
func (s *CAServiceImpl) GetCertificate(ctx context.Context, id uint) (*models.Certificate, error) {
	var cert models.Certificate
	err := s.database.DB().Preload("Node").Preload("Cluster").First(&cert, id).Error
	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}
	return &cert, nil
}

// GetCertificateBySerial retrieves a certificate by serial number
func (s *CAServiceImpl) GetCertificateBySerial(ctx context.Context, serial string) (*models.Certificate, error) {
	var cert models.Certificate
	err := s.database.DB().Preload("Node").Preload("Cluster").Where("serial_number = ?", serial).First(&cert).Error
	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}
	return &cert, nil
}

// ListCertificates lists certificates with filtering options
func (s *CAServiceImpl) ListCertificates(ctx context.Context, opts ListCertificatesOptions) ([]*models.Certificate, int64, error) {
	query := s.database.DB().Model(&models.Certificate{}).Preload("Node").Preload("Cluster")
	
	// Apply filters
	if opts.Type != nil {
		query = query.Where("type = ?", *opts.Type)
	}
	if opts.Status != nil {
		query = query.Where("status = ?", *opts.Status)
	}
	if opts.NodeID != nil {
		query = query.Where("node_id = ?", *opts.NodeID)
	}
	if opts.ClusterID != nil {
		query = query.Where("cluster_id = ?", *opts.ClusterID)
	}
	if opts.Backend != nil {
		query = query.Where("backend = ?", *opts.Backend)
	}
	
	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count certificates: %w", err)
	}
	
	// Apply pagination
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}
	
	// Order by creation date (newest first)
	query = query.Order("created_at DESC")
	
	var certificates []*models.Certificate
	if err := query.Find(&certificates).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list certificates: %w", err)
	}
	
	return certificates, total, nil
}

// ValidateCertificate validates a certificate
func (s *CAServiceImpl) ValidateCertificate(ctx context.Context, certPEM string) (*CertificateValidation, error) {
	return s.backend.ValidateCertificate(ctx, certPEM)
}

// CreateCertificateRequest creates a new certificate signing request
func (s *CAServiceImpl) CreateCertificateRequest(ctx context.Context, req *CreateCSRRequest) (*models.CertificateRequest, error) {
	s.logger.WithFields(map[string]interface{}{
		"common_name": req.CommonName,
		"type":        req.Type,
	}).Info("Creating certificate request")
	
	// Validate CSR PEM
	if _, err := s.parseCSRPEM(req.CSRPEM); err != nil {
		return nil, fmt.Errorf("invalid CSR PEM: %w", err)
	}
	
	// Create certificate request record
	csrRecord := &models.CertificateRequest{
		CommonName:     req.CommonName,
		Type:          req.Type,
		Status:        models.CSRStatusPending,
		CSRPEM:        req.CSRPEM,
		ValidityPeriod: req.ValidityPeriod,
		NodeID:        req.NodeID,
		ClusterID:     req.ClusterID,
	}
	
	// Set SANs, key usage, and extended key usage
	if len(req.SANs) > 0 {
		sansJSON, _ := json.Marshal(req.SANs)
		csrRecord.SANs = string(sansJSON)
	}
	if len(req.KeyUsage) > 0 {
		csrRecord.KeyUsage = fmt.Sprintf("[%s]", strings.Join(req.KeyUsage, ","))
	}
	if len(req.ExtKeyUsage) > 0 {
		csrRecord.ExtKeyUsage = fmt.Sprintf("[%s]", strings.Join(req.ExtKeyUsage, ","))
	}
	
	if err := s.database.DB().Create(csrRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to store certificate request: %w", err)
	}
	
	s.logger.WithField("csr_id", csrRecord.ID).Info("Certificate request created")
	return csrRecord, nil
}

// ProcessCertificateRequest processes a pending CSR
func (s *CAServiceImpl) ProcessCertificateRequest(ctx context.Context, csrID uint, approve bool) (*models.Certificate, error) {
	s.logger.WithFields(map[string]interface{}{
		"csr_id":  csrID,
		"approve": approve,
	}).Info("Processing certificate request")
	
	// Get the CSR
	var csr models.CertificateRequest
	if err := s.database.DB().First(&csr, csrID).Error; err != nil {
		return nil, fmt.Errorf("certificate request not found: %w", err)
	}
	
	if !csr.IsPending() {
		return nil, fmt.Errorf("certificate request is not pending")
	}
	
	if !approve {
		// Reject the request
		csr.Status = models.CSRStatusRejected
		processedAt := time.Now()
		csr.ProcessedAt = &processedAt
		csr.FailureReason = "Request rejected by administrator"
		
		if err := s.database.DB().Save(&csr).Error; err != nil {
			return nil, fmt.Errorf("failed to update CSR status: %w", err)
		}
		
		return nil, fmt.Errorf("certificate request was rejected")
	}
	
	// Create issuance request from CSR
	req := &IssueCertificateRequest{
		CommonName: csr.CommonName,
		Type:      csr.Type,
		NodeID:    csr.NodeID,
		ClusterID: csr.ClusterID,
		AutoRenew: true,
	}
	
	// Parse SANs
	if csr.SANs != "" {
		var sans []string
		if err := json.Unmarshal([]byte(csr.SANs), &sans); err == nil {
			req.SANs = sans
		}
	}
	
	// Parse validity period
	if csr.ValidityPeriod != "" {
		if duration, err := time.ParseDuration(csr.ValidityPeriod); err == nil {
			req.ValidityPeriod = duration
		}
	}
	
	// Issue the certificate
	cert, err := s.IssueCertificate(ctx, req)
	if err != nil {
		// Mark CSR as failed
		csr.Status = models.CSRStatusFailed
		processedAt := time.Now()
		csr.ProcessedAt = &processedAt
		csr.FailureReason = err.Error()
		s.database.DB().Save(&csr)
		
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}
	
	// Mark CSR as approved
	csr.Status = models.CSRStatusApproved
	processedAt := time.Now()
	csr.ProcessedAt = &processedAt
	csr.CertificateID = &cert.ID
	
	if err := s.database.DB().Save(&csr).Error; err != nil {
		s.logger.WithError(err).Warn("Failed to update CSR status after successful issuance")
	}
	
	s.logger.WithFields(map[string]interface{}{
		"csr_id":  csrID,
		"cert_id": cert.ID,
	}).Info("Certificate request processed successfully")
	
	return cert, nil
}

// ListCertificateRequests lists certificate signing requests
func (s *CAServiceImpl) ListCertificateRequests(ctx context.Context, opts ListCSROptions) ([]*models.CertificateRequest, int64, error) {
	query := s.database.DB().Model(&models.CertificateRequest{}).Preload("Node").Preload("Cluster").Preload("Certificate")
	
	// Apply filters
	if opts.Status != nil {
		query = query.Where("status = ?", *opts.Status)
	}
	if opts.Type != nil {
		query = query.Where("type = ?", *opts.Type)
	}
	if opts.NodeID != nil {
		query = query.Where("node_id = ?", *opts.NodeID)
	}
	if opts.ClusterID != nil {
		query = query.Where("cluster_id = ?", *opts.ClusterID)
	}
	
	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count certificate requests: %w", err)
	}
	
	// Apply pagination
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}
	
	// Order by creation date (newest first)
	query = query.Order("created_at DESC")
	
	var requests []*models.CertificateRequest
	if err := query.Find(&requests).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list certificate requests: %w", err)
	}
	
	return requests, total, nil
}

// CleanupExpiredCertificates removes expired certificates based on retention policy
func (s *CAServiceImpl) CleanupExpiredCertificates(ctx context.Context) error {
	s.logger.Info("Starting expired certificate cleanup")
	
	// Parse retention period
	retentionPeriod, err := time.ParseDuration(s.config.CertificateConfig.RetentionPeriod)
	if err != nil {
		retentionPeriod = 2160 * time.Hour // Default 90 days
	}
	
	cutoffTime := time.Now().Add(-retentionPeriod)
	
	// Find certificates that are expired and past retention period
	var expiredCerts []models.Certificate
	err = s.database.DB().Where("status = ? AND not_after < ?", models.CertificateStatusExpired, cutoffTime).Find(&expiredCerts).Error
	if err != nil {
		return fmt.Errorf("failed to find expired certificates: %w", err)
	}
	
	cleanedCount := 0
	for _, cert := range expiredCerts {
		// Delete certificate record
		if err := s.database.DB().Delete(&cert).Error; err != nil {
			s.logger.WithError(err).WithField("cert_id", cert.ID).Warn("Failed to delete expired certificate")
			continue
		}
		cleanedCount++
	}
	
	s.logger.WithFields(map[string]interface{}{
		"cleaned_count": cleanedCount,
		"cutoff_time":   cutoffTime,
	}).Info("Expired certificate cleanup completed")
	
	return nil
}

// GetCertificateStats returns certificate statistics
func (s *CAServiceImpl) GetCertificateStats(ctx context.Context) (*CertificateStats, error) {
	stats := &CertificateStats{}
	
	// Get CA info
	var err error
	stats.CAInfo, err = s.GetCAInfo(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get CA info for stats")
	}
	
	// Total certificates
	s.database.DB().Model(&models.Certificate{}).Count(&stats.TotalCertificates)
	
	// Active certificates
	s.database.DB().Model(&models.Certificate{}).Where("status = ?", models.CertificateStatusActive).Count(&stats.ActiveCertificates)
	
	// Expired certificates
	s.database.DB().Model(&models.Certificate{}).Where("status = ?", models.CertificateStatusExpired).Count(&stats.ExpiredCertificates)
	
	// Revoked certificates
	s.database.DB().Model(&models.Certificate{}).Where("status = ?", models.CertificateStatusRevoked).Count(&stats.RevokedCertificates)
	
	// Certificates issued in last 24 hours
	last24h := time.Now().Add(-24 * time.Hour)
	s.database.DB().Model(&models.Certificate{}).Where("created_at > ?", last24h).Count(&stats.CertificatesIssued24h)
	
	// Certificates expiring in next 30 days
	next30d := time.Now().Add(30 * 24 * time.Hour)
	s.database.DB().Model(&models.Certificate{}).Where("status = ? AND not_after < ?", models.CertificateStatusActive, next30d).Count(&stats.CertificatesExpiring30d)
	
	return stats, nil
}

// Helper methods

func (s *CAServiceImpl) validateCertificateRequest(req *IssueCertificateRequest) error {
	if req.CommonName == "" {
		return fmt.Errorf("common name is required")
	}
	
	if req.Type == "" {
		return fmt.Errorf("certificate type is required")
	}
	
	// Validate allowed domains if configured
	if len(s.config.CertificateConfig.AllowedDomains) > 0 {
		allowed := false
		for _, domain := range s.config.CertificateConfig.AllowedDomains {
			if matchesDomain(req.CommonName, domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("common name %s is not in allowed domains", req.CommonName)
		}
	}
	
	return nil
}

func (s *CAServiceImpl) parseCertificatePEM(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid certificate PEM")
	}
	
	return x509.ParseCertificate(block.Bytes)
}

func (s *CAServiceImpl) parseCSRPEM(csrPEM string) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, fmt.Errorf("invalid CSR PEM")
	}
	
	return x509.ParseCertificateRequest(block.Bytes)
}

func (s *CAServiceImpl) formatKeyUsage(keyUsage x509.KeyUsage) string {
	var usages []string
	if keyUsage&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "digital_signature")
	}
	if keyUsage&x509.KeyUsageContentCommitment != 0 {
		usages = append(usages, "content_commitment")
	}
	if keyUsage&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "key_encipherment")
	}
	if keyUsage&x509.KeyUsageDataEncipherment != 0 {
		usages = append(usages, "data_encipherment")
	}
	if keyUsage&x509.KeyUsageKeyAgreement != 0 {
		usages = append(usages, "key_agreement")
	}
	if keyUsage&x509.KeyUsageCertSign != 0 {
		usages = append(usages, "cert_sign")
	}
	if keyUsage&x509.KeyUsageCRLSign != 0 {
		usages = append(usages, "crl_sign")
	}
	return strings.Join(usages, ",")
}

func (s *CAServiceImpl) formatExtKeyUsage(extKeyUsage []x509.ExtKeyUsage) string {
	var usages []string
	for _, usage := range extKeyUsage {
		switch usage {
		case x509.ExtKeyUsageServerAuth:
			usages = append(usages, "server_auth")
		case x509.ExtKeyUsageClientAuth:
			usages = append(usages, "client_auth")
		case x509.ExtKeyUsageCodeSigning:
			usages = append(usages, "code_signing")
		case x509.ExtKeyUsageEmailProtection:
			usages = append(usages, "email_protection")
		case x509.ExtKeyUsageTimeStamping:
			usages = append(usages, "time_stamping")
		case x509.ExtKeyUsageOCSPSigning:
			usages = append(usages, "ocsp_signing")
		}
	}
	return strings.Join(usages, ",")
}

func (s *CAServiceImpl) updateCAStats(ctx context.Context, issued, active, revoked int) error {
	if s.backend == nil {
		return nil
	}
	
	caInfo, err := s.backend.GetCAInfo(ctx)
	if err != nil {
		return err
	}
	
	caInfo.CertificatesIssued += issued
	caInfo.CertificatesActive += active - revoked
	
	return s.database.DB().Save(caInfo).Error
}

func matchesDomain(name, pattern string) bool {
	if pattern == name {
		return true
	}
	
	// Simple wildcard matching for domains like "*.example.com"
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(name, "."+suffix) || name == suffix
	}
	
	return false
}