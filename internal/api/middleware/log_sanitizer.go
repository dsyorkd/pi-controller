package middleware

import (
	"strings"
	"unicode"
)

// sanitizeLogValue sanitizes user-provided values before logging to prevent log injection attacks.
// It removes/replaces newlines, carriage returns, and other control characters that could be used
// to inject fake log entries.
//
// Log injection attack example:
//
//	User-Agent: "Mozilla\n[ERROR] Fake log entry injected by attacker"
//
// This would appear in logs as two separate entries, potentially hiding malicious activity.
func sanitizeLogValue(value string) string {
	// Replace all newlines and carriage returns with spaces
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\t", " ")

	// Remove other control characters
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if unicode.IsControl(r) {
			builder.WriteRune(' ')
		} else {
			builder.WriteRune(r)
		}
	}

	// Trim excessive whitespace
	result := builder.String()
	result = strings.TrimSpace(result)

	// Collapse multiple spaces into one
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	// Truncate extremely long values to prevent log flooding
	const maxLogLength = 500
	if len(result) > maxLogLength {
		result = result[:maxLogLength] + "...[truncated]"
	}

	return result
}
