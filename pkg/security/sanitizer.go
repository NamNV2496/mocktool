package security

import (
	"html"
	"regexp"
	"strings"
)

var (
	// SQL/NoSQL injection patterns
	sqlInjectionPattern = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|<script|onerror|onload)`)

	// XSS patterns
	xssPattern = regexp.MustCompile(`(?i)(<script|javascript:|onerror=|onload=|<iframe|<object|<embed)`)

	// Path traversal patterns
	pathTraversalPattern = regexp.MustCompile(`\.\.\/|\.\.\\`)

	// Command injection patterns
	cmdInjectionPattern = regexp.MustCompile(`[;&|$(){}[\]<>]`)
)

// SanitizeInput sanitizes user input to prevent common injection attacks
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	// HTML escape to prevent XSS
	input = html.EscapeString(input)

	return input
}

// SanitizeJSONString sanitizes JSON string values while preserving structure
func SanitizeJSONString(input string) string {
	// For JSON values, we only escape HTML special characters
	// We don't want to break valid JSON structure
	return html.EscapeString(input)
}

// ValidateNoInjection checks if input contains potential injection patterns
func ValidateNoInjection(input string) bool {
	// Check for SQL/NoSQL injection patterns
	if sqlInjectionPattern.MatchString(input) {
		return false
	}

	// Check for XSS patterns
	if xssPattern.MatchString(input) {
		return false
	}

	// Check for path traversal
	if pathTraversalPattern.MatchString(input) {
		return false
	}

	return true
}

// ValidateNoCommandInjection checks for command injection patterns
func ValidateNoCommandInjection(input string) bool {
	return !cmdInjectionPattern.MatchString(input)
}

// SanitizePath sanitizes file paths to prevent path traversal attacks
func SanitizePath(path string) string {
	// Remove path traversal sequences
	path = pathTraversalPattern.ReplaceAllString(path, "")

	// Remove null bytes
	path = strings.ReplaceAll(path, "\x00", "")

	// Normalize slashes
	path = strings.ReplaceAll(path, "\\", "/")

	return path
}

// SanitizeHeaderValue sanitizes HTTP header values
func SanitizeHeaderValue(value string) string {
	// Remove control characters (except space and tab)
	var sanitized strings.Builder
	for _, r := range value {
		// Allow printable ASCII and common whitespace
		if (r >= 32 && r < 127) || r == '\t' {
			sanitized.WriteRune(r)
		}
	}

	// Remove CRLF injection attempts
	result := sanitized.String()
	result = strings.ReplaceAll(result, "\r", "")
	result = strings.ReplaceAll(result, "\n", "")

	// Trim and escape
	result = strings.TrimSpace(result)
	result = html.EscapeString(result)

	return result
}

// IsValidHeaderName checks if a header name is in the allowed list
func IsValidHeaderName(name string) bool {
	// Whitelist of allowed response headers
	allowedHeaders := map[string]bool{
		// Standard headers
		"content-type":              true,
		"content-length":            true,
		"cache-control":             true,
		"expires":                   true,
		"last-modified":             true,
		"etag":                      true,

		// CORS headers
		"access-control-allow-origin":      true,
		"access-control-allow-methods":     true,
		"access-control-allow-headers":     true,
		"access-control-expose-headers":    true,
		"access-control-max-age":           true,
		"access-control-allow-credentials": true,

		// Custom application headers (prefixed)
		"x-request-id":     true,
		"x-trace-id":       true,
		"x-correlation-id": true,
		"x-api-version":    true,
		"x-ratelimit-limit":     true,
		"x-ratelimit-remaining": true,
		"x-ratelimit-reset":     true,
	}

	// Normalize header name to lowercase
	normalizedName := strings.ToLower(strings.TrimSpace(name))

	// Check if it's in the whitelist
	if allowedHeaders[normalizedName] {
		return true
	}

	// Allow custom headers with specific prefixes
	if strings.HasPrefix(normalizedName, "x-custom-") {
		return true
	}

	return false
}

// ValidateAndSanitizeHeaders validates and sanitizes a map of headers
func ValidateAndSanitizeHeaders(headers map[string]string) (map[string]string, []string) {
	sanitized := make(map[string]string)
	warnings := []string{}

	for name, value := range headers {
		// Check if header name is allowed
		if !IsValidHeaderName(name) {
			warnings = append(warnings, "Header not allowed: "+name)
			continue
		}

		// Sanitize header value
		sanitizedValue := SanitizeHeaderValue(value)

		// Check if sanitization changed the value significantly
		if sanitizedValue != value {
			warnings = append(warnings, "Header value sanitized: "+name)
		}

		sanitized[name] = sanitizedValue
	}

	return sanitized, warnings
}
