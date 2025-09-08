package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// CAHandler handles Certificate Authority related API operations
type CAHandler struct {
	caService services.CAService
	logger    logger.Interface
}

// NewCAHandler creates a new CA handler
func NewCAHandler(caService services.CAService, logger logger.Interface) *CAHandler {
	return &CAHandler{
		caService: caService,
		logger:    logger.WithField("handler", "ca"),
	}
}

// Request and response types for REST API

// IssueCertificateRestRequest represents a REST API certificate issuance request
type IssueCertificateRestRequest struct {
	CommonName     string   `json:"common_name" binding:"required"`
	Type           string   `json:"type" binding:"required"`
	SANs           []string `json:"sans,omitempty"`
	ValidityPeriod string   `json:"validity_period,omitempty"`
	KeyUsage       []string `json:"key_usage,omitempty"`
	ExtKeyUsage    []string `json:"ext_key_usage,omitempty"`
	NodeID         *uint    `json:"node_id,omitempty"`
	ClusterID      *uint    `json:"cluster_id,omitempty"`
	AutoRenew      bool     `json:"auto_renew"`
}

// CreateCSRRestRequest represents a REST API CSR creation request
type CreateCSRRestRequest struct {
	CommonName     string   `json:"common_name" binding:"required"`
	Type           string   `json:"type" binding:"required"`
	CSRPEM         string   `json:"csr_pem" binding:"required"`
	SANs           []string `json:"sans,omitempty"`
	ValidityPeriod string   `json:"validity_period,omitempty"`
	KeyUsage       []string `json:"key_usage,omitempty"`
	ExtKeyUsage    []string `json:"ext_key_usage,omitempty"`
	NodeID         *uint    `json:"node_id,omitempty"`
	ClusterID      *uint    `json:"cluster_id,omitempty"`
}

// ProcessCSRRestRequest represents a REST API CSR processing request
type ProcessCSRRestRequest struct {
	Approve bool `json:"approve" binding:"required"`
}

// ValidateCertificateRestRequest represents a REST API certificate validation request
type ValidateCertificateRestRequest struct {
	CertificatePEM string `json:"certificate_pem" binding:"required"`
}

// RevokeCertificateRestRequest represents a REST API certificate revocation request
type RevokeCertificateRestRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// CA Management Endpoints

// InitializeCA initializes the Certificate Authority
func (h *CAHandler) InitializeCA(c *gin.Context) {
	ctx := c.Request.Context()

	err := h.caService.InitializeCA(ctx)
	if err != nil {
		h.handleServiceError(c, err, "Failed to initialize CA")
		return
	}

	// Get CA info after initialization
	caInfo, err := h.caService.GetCAInfo(ctx)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get CA info after initialization")
	}

	h.logger.Info("CA initialized successfully")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Certificate Authority initialized successfully",
		"ca_info": caInfo,
	})
}

// GetCAInfo returns information about the Certificate Authority
func (h *CAHandler) GetCAInfo(c *gin.Context) {
	ctx := c.Request.Context()

	caInfo, err := h.caService.GetCAInfo(ctx)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get CA info")
		return
	}

	c.JSON(http.StatusOK, caInfo)
}

// GetCACertificate returns the CA certificate
func (h *CAHandler) GetCACertificate(c *gin.Context) {
	ctx := c.Request.Context()

	caCert, err := h.caService.GetCACertificate(ctx)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get CA certificate")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"certificate_pem": string(caCert.Raw),
		"serial_number":   caCert.SerialNumber.String(),
		"subject":         caCert.Subject.String(),
		"issuer":          caCert.Issuer.String(),
		"not_before":      caCert.NotBefore,
		"not_after":       caCert.NotAfter,
	})
}

// Certificate Management Endpoints

// IssueCertificate issues a new certificate
func (h *CAHandler) IssueCertificate(c *gin.Context) {
	var req IssueCertificateRestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Convert string type to models.CertificateType
	certType, err := h.parseStringToCertificateType(req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid certificate type: " + req.Type,
		})
		return
	}

	// Parse validity period if provided
	var validityPeriod time.Duration
	if req.ValidityPeriod != "" {
		var err error
		validityPeriod, err = time.ParseDuration(req.ValidityPeriod)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Request",
				"message": "Invalid validity period format",
			})
			return
		}
	}

	// Create service request
	serviceReq := &services.IssueCertificateRequest{
		CommonName:     req.CommonName,
		Type:           certType,
		SANs:           req.SANs,
		ValidityPeriod: validityPeriod,
		KeyUsage:       req.KeyUsage,
		ExtKeyUsage:    req.ExtKeyUsage,
		NodeID:         req.NodeID,
		ClusterID:      req.ClusterID,
		AutoRenew:      req.AutoRenew,
	}

	ctx := c.Request.Context()
	certificate, err := h.caService.IssueCertificate(ctx, serviceReq)
	if err != nil {
		h.handleServiceError(c, err, "Failed to issue certificate")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"cert_id":       certificate.ID,
		"common_name":   certificate.CommonName,
		"serial_number": certificate.SerialNumber,
	}).Info("Certificate issued successfully")

	c.JSON(http.StatusCreated, certificate)
}

// GetCertificate retrieves a certificate by ID
func (h *CAHandler) GetCertificate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid certificate ID",
		})
		return
	}

	ctx := c.Request.Context()
	certificate, err := h.caService.GetCertificate(ctx, uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to get certificate")
		return
	}

	c.JSON(http.StatusOK, certificate)
}

// GetCertificateBySerial retrieves a certificate by serial number
func (h *CAHandler) GetCertificateBySerial(c *gin.Context) {
	serialNumber := c.Param("serial")
	if serialNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Serial number is required",
		})
		return
	}

	ctx := c.Request.Context()
	certificate, err := h.caService.GetCertificateBySerial(ctx, serialNumber)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get certificate")
		return
	}

	c.JSON(http.StatusOK, certificate)
}

// ListCertificates lists certificates with optional filtering
func (h *CAHandler) ListCertificates(c *gin.Context) {
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	opts := services.ListCertificatesOptions{
		Limit:  limit,
		Offset: offset,
	}

	// Parse optional filters
	if certType := c.Query("type"); certType != "" {
		if parsed, err := h.parseStringToCertificateType(certType); err == nil {
			opts.Type = &parsed
		}
	}

	if status := c.Query("status"); status != "" {
		if parsed, err := h.parseStringToCertificateStatus(status); err == nil {
			opts.Status = &parsed
		}
	}

	if backend := c.Query("backend"); backend != "" {
		if parsed, err := h.parseStringToCertificateBackend(backend); err == nil {
			opts.Backend = &parsed
		}
	}

	if nodeIDStr := c.Query("node_id"); nodeIDStr != "" {
		if nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32); err == nil {
			id := uint(nodeID)
			opts.NodeID = &id
		}
	}

	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			id := uint(clusterID)
			opts.ClusterID = &id
		}
	}

	ctx := c.Request.Context()
	certificates, total, err := h.caService.ListCertificates(ctx, opts)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list certificates")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"certificates": certificates,
		"count":        len(certificates),
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// RenewCertificate renews an existing certificate
func (h *CAHandler) RenewCertificate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid certificate ID",
		})
		return
	}

	ctx := c.Request.Context()
	certificate, err := h.caService.RenewCertificate(ctx, uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to renew certificate")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"old_cert_id": id,
		"new_cert_id": certificate.ID,
	}).Info("Certificate renewed successfully")

	c.JSON(http.StatusOK, certificate)
}

// RevokeCertificate revokes a certificate
func (h *CAHandler) RevokeCertificate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid certificate ID",
		})
		return
	}

	var req RevokeCertificateRestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	err = h.caService.RevokeCertificate(ctx, uint(id), req.Reason)
	if err != nil {
		h.handleServiceError(c, err, "Failed to revoke certificate")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"cert_id": id,
		"reason":  req.Reason,
	}).Info("Certificate revoked successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Certificate revoked successfully",
	})
}

// ValidateCertificate validates a certificate
func (h *CAHandler) ValidateCertificate(c *gin.Context) {
	var req ValidateCertificateRestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	validation, err := h.caService.ValidateCertificate(ctx, req.CertificatePEM)
	if err != nil {
		h.handleServiceError(c, err, "Failed to validate certificate")
		return
	}

	c.JSON(http.StatusOK, validation)
}

// Certificate Request (CSR) Endpoints

// CreateCertificateRequest creates a new certificate signing request
func (h *CAHandler) CreateCertificateRequest(c *gin.Context) {
	var req CreateCSRRestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Convert string type to models.CertificateType
	certType, err := h.parseStringToCertificateType(req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid certificate type: " + req.Type,
		})
		return
	}

	// Create service request
	serviceReq := &services.CreateCSRRequest{
		CommonName:     req.CommonName,
		Type:           certType,
		CSRPEM:         req.CSRPEM,
		SANs:           req.SANs,
		ValidityPeriod: req.ValidityPeriod,
		KeyUsage:       req.KeyUsage,
		ExtKeyUsage:    req.ExtKeyUsage,
		NodeID:         req.NodeID,
		ClusterID:      req.ClusterID,
	}

	ctx := c.Request.Context()
	csr, err := h.caService.CreateCertificateRequest(ctx, serviceReq)
	if err != nil {
		h.handleServiceError(c, err, "Failed to create certificate request")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"csr_id":      csr.ID,
		"common_name": csr.CommonName,
	}).Info("Certificate request created successfully")

	c.JSON(http.StatusCreated, csr)
}

// ProcessCertificateRequest processes a pending certificate signing request
func (h *CAHandler) ProcessCertificateRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid CSR ID",
		})
		return
	}

	var req ProcessCSRRestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	certificate, err := h.caService.ProcessCertificateRequest(ctx, uint(id), req.Approve)
	if err != nil {
		h.handleServiceError(c, err, "Failed to process certificate request")
		return
	}

	action := "rejected"
	if req.Approve {
		action = "approved"
	}

	h.logger.WithFields(map[string]interface{}{
		"csr_id": id,
		"action": action,
	}).Info("Certificate request processed successfully")

	if req.Approve && certificate != nil {
		c.JSON(http.StatusOK, certificate)
	} else {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Certificate request " + action + " successfully",
		})
	}
}

// ListCertificateRequests lists certificate signing requests
func (h *CAHandler) ListCertificateRequests(c *gin.Context) {
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	opts := services.ListCSROptions{
		Limit:  limit,
		Offset: offset,
	}

	// Parse optional filters
	if status := c.Query("status"); status != "" {
		if parsed, err := h.parseStringToCSRStatus(status); err == nil {
			opts.Status = &parsed
		}
	}

	if certType := c.Query("type"); certType != "" {
		if parsed, err := h.parseStringToCertificateType(certType); err == nil {
			opts.Type = &parsed
		}
	}

	if nodeIDStr := c.Query("node_id"); nodeIDStr != "" {
		if nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32); err == nil {
			id := uint(nodeID)
			opts.NodeID = &id
		}
	}

	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			id := uint(clusterID)
			opts.ClusterID = &id
		}
	}

	ctx := c.Request.Context()
	requests, total, err := h.caService.ListCertificateRequests(ctx, opts)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list certificate requests")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"requests": requests,
		"count":    len(requests),
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// CA Statistics and Maintenance Endpoints

// GetCertificateStats returns certificate statistics
func (h *CAHandler) GetCertificateStats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.caService.GetCertificateStats(ctx)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get certificate statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CleanupExpiredCertificates removes expired certificates
func (h *CAHandler) CleanupExpiredCertificates(c *gin.Context) {
	ctx := c.Request.Context()
	err := h.caService.CleanupExpiredCertificates(ctx)
	if err != nil {
		h.handleServiceError(c, err, "Failed to cleanup expired certificates")
		return
	}

	h.logger.Info("Certificate cleanup completed successfully")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Expired certificates cleaned up successfully",
	})
}

// Helper methods

func (h *CAHandler) handleServiceError(c *gin.Context, err error, message string) {
	h.logger.WithError(err).Error(message)

	// TODO: Add specific error type handling like in cluster handler
	// For now, default to internal server error
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "Internal Server Error",
		"message": message,
	})
}

func (h *CAHandler) parseStringToCertificateType(typeStr string) (models.CertificateType, error) {
	switch typeStr {
	case "ca":
		return models.CertificateTypeCA, nil
	case "server":
		return models.CertificateTypeServer, nil
	case "client":
		return models.CertificateTypeClient, nil
	case "ssh":
		return models.CertificateTypeSSH, nil
	case "intermediate":
		return models.CertificateTypeIntermediate, nil
	default:
		return "", models.ErrInvalidCertificatePEM // Reuse error for invalid type
	}
}

func (h *CAHandler) parseStringToCertificateStatus(statusStr string) (models.CertificateStatus, error) {
	switch statusStr {
	case "active":
		return models.CertificateStatusActive, nil
	case "expired":
		return models.CertificateStatusExpired, nil
	case "revoked":
		return models.CertificateStatusRevoked, nil
	case "pending":
		return models.CertificateStatusPending, nil
	case "failed":
		return models.CertificateStatusFailed, nil
	case "renewing":
		return models.CertificateStatusRenewing, nil
	default:
		return "", models.ErrInvalidCertificatePEM // Reuse error for invalid status
	}
}

func (h *CAHandler) parseStringToCertificateBackend(backendStr string) (models.CertificateBackend, error) {
	switch backendStr {
	case "local":
		return models.CertificateBackendLocal, nil
	case "vault":
		return models.CertificateBackendVault, nil
	default:
		return "", models.ErrInvalidCertificatePEM // Reuse error for invalid backend
	}
}

func (h *CAHandler) parseStringToCSRStatus(statusStr string) (models.CSRStatus, error) {
	switch statusStr {
	case "pending":
		return models.CSRStatusPending, nil
	case "approved":
		return models.CSRStatusApproved, nil
	case "rejected":
		return models.CSRStatusRejected, nil
	case "failed":
		return models.CSRStatusFailed, nil
	default:
		return "", models.ErrCSRNotFound // Reuse error for invalid status
	}
}
