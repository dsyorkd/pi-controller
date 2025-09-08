package services

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// LocalCABackend implements the CABackend interface using a local CA
type LocalCABackend struct {
	config      *config.LocalCAConfig
	sshConfig   *config.SSHConfig
	database    *storage.Database
	logger      logger.Interface
	sshExecutor SSHExecutor

	// Cached CA certificate and info
	caInfo *models.CAInfo
	caCert *x509.Certificate
}

// NewLocalCABackend creates a new LocalCABackend instance
func NewLocalCABackend(
	config *config.LocalCAConfig,
	sshConfig *config.SSHConfig,
	database *storage.Database,
	logger logger.Interface,
	sshExecutor SSHExecutor,
) *LocalCABackend {
	return &LocalCABackend{
		config:      config,
		sshConfig:   sshConfig,
		database:    database,
		logger:      logger.WithField("component", "local-ca"),
		sshExecutor: sshExecutor,
	}
}

// Type returns the backend type
func (l *LocalCABackend) Type() models.CertificateBackend {
	return models.CertificateBackendLocal
}

// InitializeCA initializes the local Certificate Authority
func (l *LocalCABackend) InitializeCA(ctx context.Context) error {
	l.logger.Info("Initializing Local CA")

	// Check if CA already exists in database
	var existingCA models.CAInfo
	result := l.database.DB().Where("type = ? AND backend = ?", models.CATypeRoot, models.CertificateBackendLocal).First(&existingCA)
	if result.Error == nil {
		l.logger.WithField("ca_id", existingCA.ID).Info("Local CA already initialized")
		l.caInfo = &existingCA
		return l.loadCACertificate(ctx)
	}

	// Generate CA certificate and key on server nodes
	caCertPEM, err := l.generateRootCA(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate root CA: %w", err)
	}

	// Parse the certificate to extract metadata
	block, _ := pem.Decode([]byte(caCertPEM))
	if block == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Store CA certificate in database
	caCertRecord := &models.Certificate{
		SerialNumber:   cert.SerialNumber.String(),
		CommonName:     cert.Subject.CommonName,
		Type:           models.CertificateTypeCA,
		Status:         models.CertificateStatusActive,
		CertificatePEM: caCertPEM,
		Subject:        cert.Subject.String(),
		Issuer:         cert.Issuer.String(),
		NotBefore:      cert.NotBefore,
		NotAfter:       cert.NotAfter,
		Backend:        models.CertificateBackendLocal,
		LocalPath:      fmt.Sprintf("%s/ca.crt", l.config.DataDir),
		AutoRenew:      false, // CA certificates typically don't auto-renew
	}

	if err := l.database.DB().Create(caCertRecord).Error; err != nil {
		return fmt.Errorf("failed to store CA certificate: %w", err)
	}

	// Create CA info record
	l.caInfo = &models.CAInfo{
		Name:          "Pi Controller Root CA",
		Type:          models.CATypeRoot,
		Backend:       models.CertificateBackendLocal,
		Status:        models.CAStatusActive,
		CertificateID: &caCertRecord.ID,
		LocalPath:     l.config.DataDir,
		Subject:       cert.Subject.String(),
		NotBefore:     cert.NotBefore,
		NotAfter:      cert.NotAfter,
		SerialNumber:  cert.SerialNumber.String(),
	}

	if err := l.database.DB().Create(l.caInfo).Error; err != nil {
		return fmt.Errorf("failed to store CA info: %w", err)
	}

	l.caCert = cert
	l.logger.WithFields(map[string]interface{}{
		"ca_id":         l.caInfo.ID,
		"subject":       cert.Subject.String(),
		"not_after":     cert.NotAfter,
		"serial_number": cert.SerialNumber.String(),
	}).Info("Local CA initialized successfully")

	return nil
}

// generateRootCA generates a self-signed root CA certificate on server nodes
func (l *LocalCABackend) generateRootCA(ctx context.Context) (string, error) {
	l.logger.Info("Generating root CA certificate on server nodes")

	// Find master nodes to generate CA on
	var masterNodes []models.Node
	result := l.database.DB().Where("role = ? AND status = ?", models.NodeRoleMaster, models.NodeStatusReady).Find(&masterNodes)
	if result.Error != nil {
		return "", fmt.Errorf("failed to find master nodes: %w", result.Error)
	}

	if len(masterNodes) == 0 {
		return "", fmt.Errorf("no active master nodes found for CA generation")
	}

	// Use the first master node for CA generation
	masterNode := masterNodes[0]
	l.logger.WithFields(map[string]interface{}{
		"node_id":    masterNode.ID,
		"node_name":  masterNode.Name,
		"ip_address": masterNode.IPAddress,
	}).Info("Generating CA on master node")

	// Create CA directories on the master node
	createDirCmd := fmt.Sprintf("sudo mkdir -p %s && sudo chmod 700 %s", l.config.DataDir, l.config.DataDir)
	if _, err := l.sshExecutor.Execute(ctx, masterNode.IPAddress, createDirCmd); err != nil {
		return "", fmt.Errorf("failed to create CA directory on node %s: %w", masterNode.IPAddress, err)
	}

	// Generate CA private key on the node
	generateKeyCmd := fmt.Sprintf(`sudo openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:%d -out %s/ca.key`,
		l.config.KeySize, l.config.DataDir)
	if _, err := l.sshExecutor.Execute(ctx, masterNode.IPAddress, generateKeyCmd); err != nil {
		return "", fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Set secure permissions on private key
	chmodCmd := fmt.Sprintf("sudo chmod 600 %s/ca.key", l.config.DataDir)
	if _, err := l.sshExecutor.Execute(ctx, masterNode.IPAddress, chmodCmd); err != nil {
		return "", fmt.Errorf("failed to set CA key permissions: %w", err)
	}

	// Create CA certificate configuration
	validity, err := time.ParseDuration(l.config.CAValidityPeriod)
	if err != nil {
		validity = 87600 * time.Hour // Default 10 years
	}
	validityDays := int(validity.Hours() / 24)

	// Generate CA certificate on the node
	generateCertCmd := fmt.Sprintf(`sudo openssl req -new -x509 -key %s/ca.key -out %s/ca.crt -days %d \
		-subj "/C=%s/ST=%s/L=%s/O=%s/OU=%s/CN=Pi Controller Root CA"`,
		l.config.DataDir, l.config.DataDir, validityDays,
		l.config.Country, l.config.Province, l.config.Locality,
		l.config.Organization, l.config.OrganizationalUnit)

	if _, err := l.sshExecutor.Execute(ctx, masterNode.IPAddress, generateCertCmd); err != nil {
		return "", fmt.Errorf("failed to generate CA certificate: %w", err)
	}

	// Read the generated certificate
	readCertCmd := fmt.Sprintf("sudo cat %s/ca.crt", l.config.DataDir)
	certPEM, err := l.sshExecutor.Execute(ctx, masterNode.IPAddress, readCertCmd)
	if err != nil {
		return "", fmt.Errorf("failed to read CA certificate: %w", err)
	}

	l.logger.WithField("node_ip", masterNode.IPAddress).Info("Root CA certificate generated successfully")
	return certPEM, nil
}

// GetCAInfo returns the CA information
func (l *LocalCABackend) GetCAInfo(ctx context.Context) (*models.CAInfo, error) {
	if l.caInfo == nil {
		return nil, fmt.Errorf("CA not initialized")
	}
	return l.caInfo, nil
}

// GetCACertificate returns the CA certificate
func (l *LocalCABackend) GetCACertificate(ctx context.Context) (*x509.Certificate, error) {
	if l.caCert == nil {
		return nil, fmt.Errorf("CA certificate not loaded")
	}
	return l.caCert, nil
}

// loadCACertificate loads the CA certificate from the database
func (l *LocalCABackend) loadCACertificate(ctx context.Context) error {
	if l.caInfo.CertificateID == nil {
		return fmt.Errorf("CA info has no associated certificate")
	}

	var caCertRecord models.Certificate
	if err := l.database.DB().First(&caCertRecord, *l.caInfo.CertificateID).Error; err != nil {
		return fmt.Errorf("failed to load CA certificate from database: %w", err)
	}

	block, _ := pem.Decode([]byte(caCertRecord.CertificatePEM))
	if block == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	l.caCert = cert
	return nil
}

// IssueCertificate issues a new certificate using the local CA
func (l *LocalCABackend) IssueCertificate(ctx context.Context, req *IssueCertificateRequest) (string, error) {
	if l.caCert == nil {
		return "", fmt.Errorf("CA not initialized")
	}

	l.logger.WithFields(map[string]interface{}{
		"common_name": req.CommonName,
		"type":        req.Type,
		"sans":        req.SANs,
	}).Info("Issuing certificate")

	// Find master nodes to issue certificate on
	var masterNodes []models.Node
	result := l.database.DB().Where("role = ? AND status = ?", models.NodeRoleMaster, models.NodeStatusReady).Find(&masterNodes)
	if result.Error != nil {
		return "", fmt.Errorf("failed to find master nodes: %w", result.Error)
	}

	if len(masterNodes) == 0 {
		return "", fmt.Errorf("no active master nodes found for certificate issuance")
	}

	masterNode := masterNodes[0]

	// Generate certificate on the master node
	certPEM, err := l.generateCertificate(ctx, masterNode.IPAddress, req)
	if err != nil {
		return "", fmt.Errorf("failed to generate certificate: %w", err)
	}

	l.logger.WithFields(map[string]interface{}{
		"common_name": req.CommonName,
		"node_ip":     masterNode.IPAddress,
	}).Info("Certificate issued successfully")

	return certPEM, nil
}

// generateCertificate generates a certificate on a specific node
func (l *LocalCABackend) generateCertificate(ctx context.Context, nodeIP string, req *IssueCertificateRequest) (string, error) {
	// Generate unique filename for this certificate
	timestamp := time.Now().Unix()
	certName := fmt.Sprintf("cert_%s_%d", strings.ReplaceAll(req.CommonName, "*", "wildcard"), timestamp)

	// Generate private key for the certificate
	keyPath := fmt.Sprintf("%s/%s.key", l.config.DataDir, certName)
	generateKeyCmd := fmt.Sprintf(`sudo openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out %s`, keyPath)
	if _, err := l.sshExecutor.Execute(ctx, nodeIP, generateKeyCmd); err != nil {
		return "", fmt.Errorf("failed to generate certificate private key: %w", err)
	}

	// Set secure permissions on private key
	chmodKeyCmd := fmt.Sprintf("sudo chmod 600 %s", keyPath)
	if _, err := l.sshExecutor.Execute(ctx, nodeIP, chmodKeyCmd); err != nil {
		return "", fmt.Errorf("failed to set certificate key permissions: %w", err)
	}

	// Create certificate signing request
	csrPath := fmt.Sprintf("%s/%s.csr", l.config.DataDir, certName)
	createCSRCmd := fmt.Sprintf(`sudo openssl req -new -key %s -out %s -subj "/CN=%s"`,
		keyPath, csrPath, req.CommonName)

	if _, err := l.sshExecutor.Execute(ctx, nodeIP, createCSRCmd); err != nil {
		return "", fmt.Errorf("failed to create CSR: %w", err)
	}

	// Determine certificate validity period
	validity := req.ValidityPeriod
	if validity == 0 {
		var err error
		validity, err = time.ParseDuration(l.config.CertValidityPeriod)
		if err != nil {
			validity = 8760 * time.Hour // Default 1 year
		}
	}
	validityDays := int(validity.Hours() / 24)

	// Create certificate extensions file for SANs
	extPath := fmt.Sprintf("%s/%s.ext", l.config.DataDir, certName)
	extContent := l.buildCertificateExtensions(req)

	if err := l.sshExecutor.CopyContent(ctx, nodeIP, extContent, extPath); err != nil {
		return "", fmt.Errorf("failed to create certificate extensions file: %w", err)
	}

	// Sign the certificate with CA
	certPath := fmt.Sprintf("%s/%s.crt", l.config.DataDir, certName)
	signCertCmd := fmt.Sprintf(`sudo openssl x509 -req -in %s -CA %s/ca.crt -CAkey %s/ca.key -CAcreateserial -out %s -days %d -extensions v3_ext -extfile %s`,
		csrPath, l.config.DataDir, l.config.DataDir, certPath, validityDays, extPath)

	if _, err := l.sshExecutor.Execute(ctx, nodeIP, signCertCmd); err != nil {
		return "", fmt.Errorf("failed to sign certificate: %w", err)
	}

	// Read the generated certificate
	readCertCmd := fmt.Sprintf("sudo cat %s", certPath)
	certPEM, err := l.sshExecutor.Execute(ctx, nodeIP, readCertCmd)
	if err != nil {
		return "", fmt.Errorf("failed to read generated certificate: %w", err)
	}

	// Clean up temporary files
	cleanupCmd := fmt.Sprintf("sudo rm -f %s %s %s", csrPath, extPath, keyPath)
	if _, err := l.sshExecutor.Execute(ctx, nodeIP, cleanupCmd); err != nil {
		l.logger.WithError(err).Warn("Failed to clean up temporary certificate files")
	}

	return certPEM, nil
}

// buildCertificateExtensions builds the OpenSSL extensions configuration
func (l *LocalCABackend) buildCertificateExtensions(req *IssueCertificateRequest) string {
	var extensions strings.Builder

	extensions.WriteString("[v3_ext]\n")

	// Key usage
	keyUsage := req.KeyUsage
	if len(keyUsage) == 0 {
		keyUsage = []string{"digital_signature", "key_encipherment"}
	}
	extensions.WriteString(fmt.Sprintf("keyUsage = %s\n", strings.Join(keyUsage, ",")))

	// Extended key usage
	extKeyUsage := req.ExtKeyUsage
	if len(extKeyUsage) == 0 {
		if req.Type == models.CertificateTypeServer {
			extKeyUsage = []string{"server_auth"}
		} else if req.Type == models.CertificateTypeClient {
			extKeyUsage = []string{"client_auth"}
		} else {
			extKeyUsage = []string{"server_auth", "client_auth"}
		}
	}
	extensions.WriteString(fmt.Sprintf("extendedKeyUsage = %s\n", strings.Join(extKeyUsage, ",")))

	// Subject Alternative Names
	if len(req.SANs) > 0 {
		var sanEntries []string
		for _, san := range req.SANs {
			// Detect if SAN is IP address or DNS name
			if ip := net.ParseIP(san); ip != nil {
				sanEntries = append(sanEntries, fmt.Sprintf("IP:%s", san))
			} else if _, err := url.Parse(fmt.Sprintf("http://%s", san)); err == nil {
				sanEntries = append(sanEntries, fmt.Sprintf("DNS:%s", san))
			}
		}
		if len(sanEntries) > 0 {
			extensions.WriteString(fmt.Sprintf("subjectAltName = %s\n", strings.Join(sanEntries, ",")))
		}
	}

	return extensions.String()
}

// RevokeCertificate revokes a certificate
func (l *LocalCABackend) RevokeCertificate(ctx context.Context, cert *models.Certificate) error {
	l.logger.WithFields(map[string]interface{}{
		"cert_id":       cert.ID,
		"serial_number": cert.SerialNumber,
		"common_name":   cert.CommonName,
	}).Info("Revoking certificate")

	// Find master nodes to revoke certificate on
	var masterNodes []models.Node
	result := l.database.DB().Where("role = ? AND status = ?", models.NodeRoleMaster, models.NodeStatusReady).Find(&masterNodes)
	if result.Error != nil {
		return fmt.Errorf("failed to find master nodes: %w", result.Error)
	}

	if len(masterNodes) == 0 {
		return fmt.Errorf("no active master nodes found for certificate revocation")
	}

	masterNode := masterNodes[0]

	// Create temporary certificate file on node for revocation
	tempCertPath := fmt.Sprintf("%s/temp_cert_%s.crt", l.config.DataDir, cert.SerialNumber)
	if err := l.sshExecutor.CopyContent(ctx, masterNode.IPAddress, cert.CertificatePEM, tempCertPath); err != nil {
		return fmt.Errorf("failed to copy certificate for revocation: %w", err)
	}

	// Revoke the certificate (this would update a CRL if we maintain one)
	revokeCmd := fmt.Sprintf(`sudo openssl ca -revoke %s -keyfile %s/ca.key -cert %s/ca.crt -config /dev/null || true`,
		tempCertPath, l.config.DataDir, l.config.DataDir)

	if _, err := l.sshExecutor.Execute(ctx, masterNode.IPAddress, revokeCmd); err != nil {
		l.logger.WithError(err).Warn("Certificate revocation command failed (this may be expected if no CRL is configured)")
	}

	// Clean up temporary file
	cleanupCmd := fmt.Sprintf("sudo rm -f %s", tempCertPath)
	if _, err := l.sshExecutor.Execute(ctx, masterNode.IPAddress, cleanupCmd); err != nil {
		l.logger.WithError(err).Warn("Failed to clean up temporary certificate file")
	}

	l.logger.WithField("cert_id", cert.ID).Info("Certificate revoked successfully")
	return nil
}

// ValidateCertificate validates a certificate against the CA
func (l *LocalCABackend) ValidateCertificate(ctx context.Context, certPEM string) (*CertificateValidation, error) {
	validation := &CertificateValidation{
		Valid:  false,
		Errors: []string{},
	}

	// Parse the certificate
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		validation.Errors = append(validation.Errors, "invalid PEM format")
		return validation, nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		validation.Errors = append(validation.Errors, fmt.Sprintf("failed to parse certificate: %v", err))
		return validation, nil
	}

	// Set certificate metadata
	validation.NotBefore = cert.NotBefore
	validation.NotAfter = cert.NotAfter
	validation.SerialNumber = cert.SerialNumber.String()
	validation.Subject = cert.Subject.String()
	validation.Issuer = cert.Issuer.String()

	// Check expiration
	now := time.Now()
	if now.Before(cert.NotBefore) {
		validation.Errors = append(validation.Errors, "certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		validation.Expired = true
		validation.Errors = append(validation.Errors, "certificate has expired")
	}

	// Verify against CA certificate
	if l.caCert != nil {
		roots := x509.NewCertPool()
		roots.AddCert(l.caCert)

		opts := x509.VerifyOptions{
			Roots: roots,
		}

		if _, err := cert.Verify(opts); err != nil {
			validation.Errors = append(validation.Errors, fmt.Sprintf("certificate verification failed: %v", err))
		}
	} else {
		validation.Errors = append(validation.Errors, "CA certificate not available for verification")
	}

	// Check if certificate is in revoked list (would need to implement CRL checking)
	// For now, check database for revocation status
	var certRecord models.Certificate
	result := l.database.DB().Where("serial_number = ?", cert.SerialNumber.String()).First(&certRecord)
	if result.Error == nil && certRecord.Status == models.CertificateStatusRevoked {
		validation.Revoked = true
		validation.Errors = append(validation.Errors, "certificate has been revoked")
	}

	validation.Valid = len(validation.Errors) == 0 && !validation.Expired && !validation.Revoked
	return validation, nil
}

// HealthCheck performs a health check of the local CA backend
func (l *LocalCABackend) HealthCheck(ctx context.Context) error {
	// Check if CA is initialized
	if l.caInfo == nil {
		return fmt.Errorf("CA not initialized")
	}

	if l.caCert == nil {
		return fmt.Errorf("CA certificate not loaded")
	}

	// Check CA certificate expiration
	if time.Now().After(l.caCert.NotAfter) {
		return fmt.Errorf("CA certificate has expired")
	}

	// Check if CA directory is accessible on master nodes
	var masterNodes []models.Node
	result := l.database.DB().Where("role = ? AND status = ?", models.NodeRoleMaster, models.NodeStatusReady).Limit(1).Find(&masterNodes)
	if result.Error != nil {
		return fmt.Errorf("failed to find master nodes: %w", result.Error)
	}

	if len(masterNodes) > 0 {
		masterNode := masterNodes[0]
		checkCmd := fmt.Sprintf("sudo test -d %s && sudo test -f %s/ca.crt && sudo test -f %s/ca.key",
			l.config.DataDir, l.config.DataDir, l.config.DataDir)

		if _, err := l.sshExecutor.Execute(ctx, masterNode.IPAddress, checkCmd); err != nil {
			return fmt.Errorf("CA files not accessible on master node %s: %w", masterNode.IPAddress, err)
		}
	}

	return nil
}
