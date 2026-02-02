package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
)

// Config holds TLS configuration
type Config struct {
	CertFile  string
	KeyFile   string
	AutoCert  bool // Auto-generate self-signed cert for development
	CertDir   string
	Hostnames []string
}

// Setup configures TLS based on the provided configuration
func Setup(cfg Config, log logger.Interface) (*tls.Config, error) {
	// If cert files are provided and exist, use them
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		if fileExists(cfg.CertFile) && fileExists(cfg.KeyFile) {
			log.WithFields(map[string]interface{}{
				"cert_file": cfg.CertFile,
				"key_file":  cfg.KeyFile,
			}).Info("Using existing TLS certificates")

			cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
			}

			return &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12, // Require TLS 1.2 or higher
				CipherSuites: getSecureCipherSuites(),
			}, nil
		}

		// Files specified but don't exist
		if !cfg.AutoCert {
			return nil, fmt.Errorf("TLS certificate files not found: cert=%s, key=%s", cfg.CertFile, cfg.KeyFile)
		}
	}

	// Auto-generate self-signed certificate for development
	if cfg.AutoCert {
		log.Warn("Auto-generating self-signed TLS certificate for development")
		log.Warn("⚠️  DO NOT use auto-generated certificates in production!")

		certFile, keyFile, err := generateSelfSignedCert(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}

		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load auto-generated certificates: %w", err)
		}

		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
			CipherSuites: getSecureCipherSuites(),
		}, nil
	}

	return nil, fmt.Errorf("TLS enabled but no certificates configured")
}

// generateSelfSignedCert creates a self-signed certificate for development
func generateSelfSignedCert(cfg Config, log logger.Interface) (string, string, error) {
	// Set default cert directory if not specified
	certDir := cfg.CertDir
	if certDir == "" {
		certDir = "./data/tls"
	}

	// Ensure cert directory exists
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create cert directory: %w", err)
	}

	certFile := filepath.Join(certDir, "server.crt")
	keyFile := filepath.Join(certDir, "server.key")

	// Check if cert already exists and is still valid
	if fileExists(certFile) && fileExists(keyFile) {
		if isCertValid(certFile, log) {
			log.WithFields(map[string]interface{}{
				"cert_file": certFile,
				"key_file":  keyFile,
			}).Info("Using existing auto-generated TLS certificate")
			return certFile, keyFile, nil
		}
		log.Info("Existing certificate expired or invalid, generating new one")
	}

	// Generate new ECDSA private key (more efficient than RSA)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Set up certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Default hostnames
	hostnames := cfg.Hostnames
	if len(hostnames) == 0 {
		hostnames = []string{"localhost", "127.0.0.1", "::1"}
	}

	// Parse IP addresses from hostnames
	var ipAddresses []net.IP
	var dnsNames []string
	for _, hostname := range hostnames {
		if ip := net.ParseIP(hostname); ip != nil {
			ipAddresses = append(ipAddresses, ip)
		} else {
			dnsNames = append(dnsNames, hostname)
		}
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"Pi Controller Development"},
			CommonName:    "Pi Controller Dev Certificate",
			Country:       []string{"US"},
			Province:      []string{"CA"},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{},
			PostalCode:    []string{},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certOut, err := os.Create(certFile) // #nosec G304 - certFile path from config, admin-controlled
	if err != nil {
		return "", "", fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", "", fmt.Errorf("failed to write cert: %w", err)
	}

	// Write private key to file
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600) // #nosec G304 - keyFile path from config, admin-controlled
	if err != nil {
		return "", "", fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return "", "", fmt.Errorf("failed to write key: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"cert_file": certFile,
		"key_file":  keyFile,
		"hostnames": hostnames,
		"valid_for": "365 days",
	}).Info("Generated self-signed TLS certificate")

	return certFile, keyFile, nil
}

// isCertValid checks if a certificate file is valid and not expired
func isCertValid(certFile string, log logger.Interface) bool {
	certPEM, err := os.ReadFile(certFile) // #nosec G304 - certFile path from config, admin-controlled
	if err != nil {
		return false
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}

	// Check if certificate is expired or will expire within 30 days
	now := time.Now()
	if now.After(cert.NotAfter) {
		log.WithField("expiry", cert.NotAfter).Info("Certificate expired")
		return false
	}

	if now.Add(30 * 24 * time.Hour).After(cert.NotAfter) {
		log.WithField("expiry", cert.NotAfter).Warn("Certificate will expire within 30 days")
	}

	return true
}

// getSecureCipherSuites returns a list of secure cipher suites
func getSecureCipherSuites() []uint16 {
	return []uint16{
		// TLS 1.3 cipher suites (automatically used if available)
		// TLS 1.2 cipher suites
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetTLSConfigForClient returns a TLS config for client connections
func GetTLSConfigForClient(insecure bool, caCertFile string) (*tls.Config, error) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if insecure {
		config.InsecureSkipVerify = true
		return config, nil
	}

	if caCertFile != "" {
		caCert, err := os.ReadFile(caCertFile) // #nosec G304 - caCertFile path from config, admin-controlled
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}

		config.RootCAs = caCertPool
	}

	return config, nil
}

// ValidateTLSConfig validates TLS configuration
func ValidateTLSConfig(certFile, keyFile string) error {
	if certFile == "" {
		return fmt.Errorf("TLS certificate file not specified")
	}
	if keyFile == "" {
		return fmt.Errorf("TLS key file not specified")
	}

	// Try to load the certificate
	_, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("invalid TLS configuration: %w", err)
	}

	return nil
}
