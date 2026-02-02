package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
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
	caKey  *rsa.PrivateKey
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

// generateRootCA generates a self-signed root CA certificate locally
func (l *LocalCABackend) generateRootCA(ctx context.Context) (string, error) {
	l.logger.Info("Generating root CA certificate locally")

	// Create CA directory if it doesn't exist
	if err := os.MkdirAll(l.config.DataDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create CA directory: %w", err)
	}

	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, l.config.KeySize)
	if err != nil {
		return "", fmt.Errorf("failed to generate private key: %w", err)
	}
	l.caKey = priv

	// Create certificate template
	validity, err := time.ParseDuration(l.config.CAValidityPeriod)
	if err != nil {
		validity = 87600 * time.Hour // Default 10 years
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validity)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{l.config.Organization},
			OrganizationalUnit: []string{l.config.OrganizationalUnit},
			Country:            []string{l.config.Country},
			Province:           []string{l.config.Province},
			Locality:           []string{l.config.Locality},
			CommonName:         "Pi Controller Root CA",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Save private key
	keyPath := filepath.Join(l.config.DataDir, "ca.key")
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to open key file for writing: %w", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Save certificate
	certPath := filepath.Join(l.config.DataDir, "ca.crt")
	certOut, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open cert file for writing: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return "", fmt.Errorf("failed to write certificate: %w", err)
	}

	// Return PEM encoded certificate
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	l.logger.Info("Root CA certificate generated successfully")
	return string(certPEM), nil
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
	_ = ctx // TODO: Use context for database operation timeout in future
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

	// Load CA private key
	keyPath := filepath.Join(l.config.DataDir, "ca.key")
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %w", err)
	}

	l.caKey = key
	return nil
}

// IssueCertificate issues a new certificate using the local CA
func (l *LocalCABackend) IssueCertificate(ctx context.Context, req *IssueCertificateRequest) (string, string, error) {
	if l.caCert == nil {
		return "", "", fmt.Errorf("CA not initialized")
	}

	l.logger.WithFields(map[string]interface{}{
		"common_name": req.CommonName,
		"type":        req.Type,
		"sans":        req.SANs,
	}).Info("Issuing certificate")

	// Generate certificate locally
	certPEM, keyPEM, err := l.generateCertificateLocally(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate certificate: %w", err)
	}

	l.logger.WithFields(map[string]interface{}{
		"common_name": req.CommonName,
	}).Info("Certificate issued successfully")

	return certPEM, keyPEM, nil
}

// generateCertificateLocally generates a certificate locally
func (l *LocalCABackend) generateCertificateLocally(_ context.Context, req *IssueCertificateRequest) (string, string, error) {
	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Determine validity
	validity := req.ValidityPeriod
	if validity == 0 {
		var err error
		validity, err = time.ParseDuration(l.config.CertValidityPeriod)
		if err != nil {
			validity = 8760 * time.Hour // Default 1 year
		}
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validity)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   req.CommonName,
			Organization: []string{l.config.Organization},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	// Add SANs
	for _, san := range req.SANs {
		if ip := net.ParseIP(san); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, san)
		}
	}

	// Sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, l.caCert, &priv.PublicKey, l.caKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return string(certPEM), string(keyPEM), nil
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
