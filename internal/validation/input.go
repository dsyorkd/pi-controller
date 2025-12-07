package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"unicode"
)

// Common SQL injection patterns and dangerous characters
var (
	// SQL keywords that shouldn't appear in resource names
	sqlKeywords = []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER",
		"TRUNCATE", "EXEC", "EXECUTE", "UNION", "FROM", "WHERE", "TABLE",
	}

	// Dangerous characters for SQL injection
	sqlDangerousChars = []rune{';', '\'', '"', '\\', '\x00', '\n', '\r', '\x1a'}

	// Script injection patterns
	scriptPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`), // Event handlers like onclick=
	}

	// Path traversal patterns
	pathTraversalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\.\.[\\/]`),
		regexp.MustCompile(`[\\/]\.\.`),
	}

	// Valid resource name pattern (alphanumeric, hyphens, underscores)
	validResourceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string
	Message string
	Value   string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// ValidateResourceName validates a resource name (cluster, node, etc.) against malicious input
func ValidateResourceName(fieldName, value string, maxLength int) error {
	if value == "" {
		return &ValidationError{
			Field:   fieldName,
			Message: "name cannot be empty",
			Value:   value,
		}
	}

	if len(value) > maxLength {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("name exceeds maximum length of %d characters", maxLength),
			Value:   value,
		}
	}

	// Check for SQL injection patterns
	if err := checkSQLInjection(fieldName, value); err != nil {
		return err
	}

	// Check for script injection patterns
	if err := checkScriptInjection(fieldName, value); err != nil {
		return err
	}

	// Check for path traversal patterns
	if err := checkPathTraversal(fieldName, value); err != nil {
		return err
	}

	// Check for control characters
	if err := checkControlCharacters(fieldName, value); err != nil {
		return err
	}

	// Validate against allowed character set
	if !validResourceNamePattern.MatchString(value) {
		return &ValidationError{
			Field:   fieldName,
			Message: "name can only contain alphanumeric characters, hyphens, and underscores",
			Value:   value,
		}
	}

	return nil
}

// ValidateDescription validates a description field
func ValidateDescription(fieldName, value string, maxLength int) error {
	if len(value) > maxLength {
		return &ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("description exceeds maximum length of %d characters", maxLength),
			Value:   value,
		}
	}

	// Check for SQL injection patterns
	if err := checkSQLInjection(fieldName, value); err != nil {
		return err
	}

	// Check for script injection patterns
	if err := checkScriptInjection(fieldName, value); err != nil {
		return err
	}

	// Check for control characters (except newlines and tabs)
	for _, ch := range value {
		if unicode.IsControl(ch) && ch != '\n' && ch != '\r' && ch != '\t' {
			return &ValidationError{
				Field:   fieldName,
				Message: "contains illegal control characters",
				Value:   value,
			}
		}
	}

	return nil
}

// checkSQLInjection checks for SQL injection patterns
func checkSQLInjection(fieldName, value string) error {
	upperValue := strings.ToUpper(value)

	// Check for dangerous characters first - these are always suspicious
	for _, ch := range sqlDangerousChars {
		if strings.ContainsRune(value, ch) {
			return &ValidationError{
				Field:   fieldName,
				Message: "contains characters that could be used for SQL injection",
				Value:   value,
			}
		}
	}

	// Check for SQL comment patterns
	if strings.Contains(value, "--") || strings.Contains(value, "/*") || strings.Contains(value, "*/") {
		return &ValidationError{
			Field:   fieldName,
			Message: "contains SQL comment patterns which are not allowed",
			Value:   value,
		}
	}

	// Only flag SQL keywords if they appear with suspicious patterns
	// This prevents false positives on normal text like "Updated description"
	for _, keyword := range sqlKeywords {
		// Check for keyword at word boundaries or with suspicious surrounding characters
		if containsSQLKeywordPattern(upperValue, keyword) {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("contains SQL keyword '%s' which is not allowed", keyword),
				Value:   value,
			}
		}
	}

	return nil
}

// containsSQLKeywordPattern checks if a SQL keyword appears in a suspicious context
func containsSQLKeywordPattern(upperValue, keyword string) bool {
	// Look for keyword with SQL operators or at string boundaries
	suspiciousPatterns := []string{
		" " + keyword + " ", // Keyword with spaces (common in SQL)
		"(" + keyword + " ", // After parenthesis
		" " + keyword + "(", // Before parenthesis
		";" + keyword,       // After semicolon
		keyword + ";",       // Before semicolon
		"\t" + keyword,      // After tab
		keyword + "\t",      // Before tab
		"\n" + keyword,      // After newline
		keyword + "\n",      // Before newline
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(upperValue, pattern) {
			return true
		}
	}

	// Check if it's the entire value (standalone keyword)
	return strings.TrimSpace(upperValue) == keyword
}

// checkScriptInjection checks for script injection patterns
func checkScriptInjection(fieldName, value string) error {
	for _, pattern := range scriptPatterns {
		if pattern.MatchString(value) {
			return &ValidationError{
				Field:   fieldName,
				Message: "contains script injection patterns which are not allowed",
				Value:   value,
			}
		}
	}
	return nil
}

// checkPathTraversal checks for path traversal patterns
func checkPathTraversal(fieldName, value string) error {
	for _, pattern := range pathTraversalPatterns {
		if pattern.MatchString(value) {
			return &ValidationError{
				Field:   fieldName,
				Message: "contains path traversal patterns which are not allowed",
				Value:   value,
			}
		}
	}
	return nil
}

// checkControlCharacters checks for control characters
func checkControlCharacters(fieldName, value string) error {
	for _, ch := range value {
		if unicode.IsControl(ch) {
			return &ValidationError{
				Field:   fieldName,
				Message: "contains illegal control characters",
				Value:   value,
			}
		}
	}
	return nil
}

// ValidateIPAddress validates an IP address (IPv4 or IPv6)
func ValidateIPAddress(fieldName, value string) error {
	if value == "" {
		return &ValidationError{
			Field:   fieldName,
			Message: "IP address cannot be empty",
			Value:   value,
		}
	}

	// Use net.ParseIP for robust IP validation - this prevents command injection
	// as it only accepts valid IP address formats
	if net.ParseIP(value) == nil {
		return &ValidationError{
			Field:   fieldName,
			Message: "not a valid IP address",
			Value:   value,
		}
	}

	return nil
}

// ValidateHostname validates a hostname against command injection and ensures RFC-compliant format
func ValidateHostname(fieldName, value string) error {
	if value == "" {
		return nil // Hostname is typically optional
	}

	if len(value) > 253 {
		return &ValidationError{
			Field:   fieldName,
			Message: "hostname exceeds maximum length of 253 characters",
			Value:   value,
		}
	}

	// Check for command injection patterns
	// Shell command substitution patterns
	commandInjectionPatterns := []string{
		"$(", // Command substitution $(...)
		"`",  // Backtick command substitution
		"${", // Variable expansion (can be dangerous in some contexts)
		"|",  // Pipe
		"&",  // Background execution or AND
		";",  // Command separator
		"<",  // Input redirection
		">",  // Output redirection
		"\n", // Newline (can be used for command chaining)
		"\r", // Carriage return
	}

	for _, pattern := range commandInjectionPatterns {
		if strings.Contains(value, pattern) {
			return &ValidationError{
				Field:   fieldName,
				Message: "contains characters that could be used for command injection",
				Value:   value,
			}
		}
	}

	// RFC 1123 hostname validation: labels separated by dots
	// Each label: 1-63 chars, alphanumeric or hyphens, cannot start/end with hyphen
	hostnamePattern := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	if !hostnamePattern.MatchString(value) {
		return &ValidationError{
			Field:   fieldName,
			Message: "not a valid hostname (must follow RFC 1123 format: alphanumeric and hyphens, labels separated by dots)",
			Value:   value,
		}
	}

	return nil
}

// SanitizeString removes potentially dangerous characters from a string
// This should be used as a last resort - proper validation is preferred
func SanitizeString(value string) string {
	// Remove SQL dangerous characters
	for _, ch := range sqlDangerousChars {
		value = strings.ReplaceAll(value, string(ch), "")
	}

	// Remove control characters except newlines and tabs
	var sanitized strings.Builder
	for _, ch := range value {
		if !unicode.IsControl(ch) || ch == '\n' || ch == '\r' || ch == '\t' {
			sanitized.WriteRune(ch)
		}
	}

	return sanitized.String()
}
