package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/dsyorkd/pi-controller/internal/logger"
)

var (
	// Safe name pattern: alphanumeric, hyphens, underscores, dots
	safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	// Safe IP pattern for IPv4
	ipv4Pattern = regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	// Safe hostname pattern
	hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
)

// ValidationConfig holds validation configuration
type ValidationConfig struct {
	MaxNameLength    int   `yaml:"max_name_length"`
	MaxDescLength    int   `yaml:"max_description_length"`
	MaxQueryLimit    int   `yaml:"max_query_limit"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	EnableSQLCheck   bool  `yaml:"enable_sql_check"`
	EnableXSSCheck   bool  `yaml:"enable_xss_check"`
}

// DefaultValidationConfig returns secure validation defaults
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxNameLength:  63,   // DNS-safe length
		MaxDescLength:  255,  // Reasonable description length
		MaxQueryLimit:  1000, // Prevent excessive queries
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		EnableSQLCheck: true,
		EnableXSSCheck: true,
	}
}

// Validator provides request validation middleware
type Validator struct {
	config *ValidationConfig
	logger logger.Interface
}

// NewValidator creates a new request validator
func NewValidator(config *ValidationConfig, logger logger.Interface) *Validator {
	if config == nil {
		config = DefaultValidationConfig()
	}

	return &Validator{
		config: config,
		logger: logger.WithField("component", "validator"),
	}
}

// ValidateRequest provides comprehensive request validation
func (v *Validator) ValidateRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check request method
		if !v.isAllowedMethod(c.Request.Method) {
			v.logger.WithField("method", c.Request.Method).Warn("Invalid HTTP method")
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error":   "Method Not Allowed",
				"message": "HTTP method not allowed",
			})
			c.Abort()
			return
		}

		// Validate request headers
		if err := v.validateHeaders(c); err != nil {
			v.logger.WithError(err).Warn("Invalid request headers")
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Request",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate query parameters
		if err := v.validateQueryParams(c); err != nil {
			v.logger.WithError(err).Warn("Invalid query parameters")
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Request", 
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateName validates resource names (clusters, nodes, etc.)
func (v *Validator) ValidateName(name string, resourceType string) error {
	if name == "" {
		return fmt.Errorf("%s name is required", resourceType)
	}

	if len(name) > v.config.MaxNameLength {
		return fmt.Errorf("%s name exceeds maximum length of %d characters", resourceType, v.config.MaxNameLength)
	}

	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("%s name contains invalid characters. Only alphanumeric, dots, hyphens and underscores allowed", resourceType)
	}

	// Check for SQL injection patterns
	if v.config.EnableSQLCheck && v.containsSQLInjection(name) {
		return fmt.Errorf("%s name contains potentially malicious content", resourceType)
	}

	return nil
}

// ValidateDescription validates description fields
func (v *Validator) ValidateDescription(desc string) error {
	if len(desc) > v.config.MaxDescLength {
		return fmt.Errorf("description exceeds maximum length of %d characters", v.config.MaxDescLength)
	}

	// Check for XSS patterns
	if v.config.EnableXSSCheck && v.containsXSS(desc) {
		return fmt.Errorf("description contains potentially malicious content")
	}

	// Check for SQL injection patterns
	if v.config.EnableSQLCheck && v.containsSQLInjection(desc) {
		return fmt.Errorf("description contains potentially malicious content")
	}

	return nil
}

// ValidateHostname validates hostname/IP input
func (v *Validator) ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}

	// Check if it's an IPv4 address
	if ipv4Pattern.MatchString(hostname) {
		return nil // Valid IPv4
	}

	// Check if it's a valid hostname
	if !hostnamePattern.MatchString(hostname) {
		return fmt.Errorf("invalid hostname format")
	}

	if len(hostname) > 253 {
		return fmt.Errorf("hostname too long")
	}

	return nil
}

// ValidateGPIOPin validates GPIO pin numbers
func (v *Validator) ValidateGPIOPin(pin int) error {
	// Raspberry Pi GPIO pin range validation
	if pin < 0 || pin > 27 {
		return fmt.Errorf("GPIO pin %d is outside valid range (0-27)", pin)
	}

	// Check system critical pins
	systemPins := []int{0, 1, 14, 15} // I2C, UART
	for _, systemPin := range systemPins {
		if pin == systemPin {
			return fmt.Errorf("GPIO pin %d is a system-critical pin and cannot be controlled", pin)
		}
	}

	return nil
}

// ValidatePort validates network port numbers
func (v *Validator) ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is outside valid range (1-65535)", port)
	}

	// Check for privileged ports (< 1024) in production
	if port < 1024 {
		v.logger.WithField("port", port).Warn("Using privileged port")
	}

	return nil
}

// validateHeaders validates HTTP headers for security issues
func (v *Validator) validateHeaders(c *gin.Context) error {
	// Check Content-Type for POST/PUT requests
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		contentType := c.GetHeader("Content-Type")
		if contentType != "" && !strings.HasPrefix(contentType, "application/json") && !strings.HasPrefix(contentType, "multipart/form-data") {
			return fmt.Errorf("unsupported content type: %s", contentType)
		}
	}

	// Validate User-Agent (block empty or suspicious ones)
	userAgent := c.GetHeader("User-Agent")
	if userAgent == "" {
		v.logger.Warn("Request with empty User-Agent")
	}

	return nil
}

// validateQueryParams validates query parameters
func (v *Validator) validateQueryParams(c *gin.Context) error {
	// Validate common query parameters
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return fmt.Errorf("invalid limit parameter: must be a number")
		}
		if limit < 0 || limit > v.config.MaxQueryLimit {
			return fmt.Errorf("limit parameter outside valid range (0-%d)", v.config.MaxQueryLimit)
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return fmt.Errorf("invalid offset parameter: must be a number")
		}
		if offset < 0 {
			return fmt.Errorf("offset parameter must be non-negative")
		}
	}

	// Check all query parameters for potential injection
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			if v.config.EnableSQLCheck && v.containsSQLInjection(value) {
				return fmt.Errorf("query parameter %s contains potentially malicious content", key)
			}
			if v.config.EnableXSSCheck && v.containsXSS(value) {
				return fmt.Errorf("query parameter %s contains potentially malicious content", key)
			}
		}
	}

	return nil
}

// isAllowedMethod checks if HTTP method is allowed
func (v *Validator) isAllowedMethod(method string) bool {
	for _, allowed := range v.config.AllowedMethods {
		if method == allowed {
			return true
		}
	}
	return false
}

// containsSQLInjection checks for SQL injection patterns
func (v *Validator) containsSQLInjection(input string) bool {
	lowerInput := strings.ToLower(input)
	
	// Common SQL injection patterns
	sqlPatterns := []string{
		"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_",
		"union", "select", "insert", "update", "delete", "drop", "create", "alter",
		"exec", "execute", "script", "javascript:", "vbscript:", "onload", "onerror",
	}
	
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	
	return false
}

// containsXSS checks for XSS patterns
func (v *Validator) containsXSS(input string) bool {
	lowerInput := strings.ToLower(input)
	
	// Common XSS patterns
	xssPatterns := []string{
		"<script", "</script>", "javascript:", "vbscript:", "onload=", "onerror=", 
		"onmouseover=", "onfocus=", "onblur=", "onchange=", "onsubmit=",
		"<iframe", "<object", "<embed", "<form", "eval(", "alert(",
	}
	
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	
	return false
}

// SanitizeInput removes potentially dangerous characters from input
func (v *Validator) SanitizeInput(input string) string {
	// Remove control characters
	input = strings.Map(func(r rune) rune {
		if r < 32 && r != 9 && r != 10 && r != 13 { // Allow tab, LF, CR
			return -1
		}
		return r
	}, input)
	
	// Trim whitespace
	return strings.TrimSpace(input)
}