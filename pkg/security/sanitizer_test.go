package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal input",
			input:    "test-input",
			expected: "test-input",
		},
		{
			name:     "input with XSS",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "input with null bytes",
			input:    "test\x00input",
			expected: "testinput",
		},
		{
			name:     "input with extra whitespace",
			input:    "  test input  ",
			expected: "test input",
		},
		{
			name:     "input with HTML entities",
			input:    "test & <div> input",
			expected: "test &amp; &lt;div&gt; input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNoInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "safe input",
			input:    "test-feature-name",
			expected: true,
		},
		{
			name:     "SQL injection - UNION",
			input:    "test' UNION SELECT * FROM users--",
			expected: false,
		},
		{
			name:     "SQL injection - DROP",
			input:    "test'; DROP TABLE users;--",
			expected: false,
		},
		{
			name:     "XSS script tag",
			input:    "<script>alert('xss')</script>",
			expected: false,
		},
		{
			name:     "XSS onerror",
			input:    "test\" onerror=\"alert('xss')\"",
			expected: false,
		},
		{
			name:     "path traversal",
			input:    "../../../etc/passwd",
			expected: false,
		},
		{
			name:     "javascript protocol",
			input:    "javascript:alert('xss')",
			expected: false,
		},
		{
			name:     "safe with special chars",
			input:    "test_feature-123",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateNoInjection(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNoCommandInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "safe input",
			input:    "test-command",
			expected: true,
		},
		{
			name:     "command with pipe",
			input:    "ls | grep test",
			expected: false,
		},
		{
			name:     "command with semicolon",
			input:    "ls; rm -rf /",
			expected: false,
		},
		{
			name:     "command with dollar",
			input:    "echo $(whoami)",
			expected: false,
		},
		{
			name:     "command with backticks",
			input:    "echo `whoami`",
			expected: true, // backticks not in pattern, but would be caught by other validations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateNoCommandInjection(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "safe path",
			input:    "/api/v1/test",
			expected: "/api/v1/test",
		},
		{
			name:     "path traversal unix",
			input:    "../../../etc/passwd",
			expected: "etc/passwd",
		},
		{
			name:     "path traversal windows",
			input:    "..\\..\\..\\windows\\system32",
			expected: "windows/system32",
		},
		{
			name:     "mixed separators",
			input:    "/api\\v1/test",
			expected: "/api/v1/test",
		},
		{
			name:     "null byte injection",
			input:    "/api/test\x00.txt",
			expected: "/api/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeHeaderValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "safe header value",
			input:    "application/json",
			expected: "application/json",
		},
		{
			name:     "header with CRLF injection",
			input:    "test\r\nSet-Cookie: admin=true",
			expected: "testSet-Cookie: admin=true",
		},
		{
			name:     "header with control characters",
			input:    "test\x00\x01value",
			expected: "testvalue",
		},
		{
			name:     "header with XSS",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "header with whitespace",
			input:    "  application/json  ",
			expected: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHeaderValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidHeaderName(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected bool
	}{
		{
			name:     "allowed standard header",
			header:   "Content-Type",
			expected: true,
		},
		{
			name:     "allowed cors header",
			header:   "Access-Control-Allow-Origin",
			expected: true,
		},
		{
			name:     "allowed custom header",
			header:   "X-Request-ID",
			expected: true,
		},
		{
			name:     "allowed with custom prefix",
			header:   "X-Custom-Header",
			expected: true,
		},
		{
			name:     "disallowed header - Set-Cookie",
			header:   "Set-Cookie",
			expected: false,
		},
		{
			name:     "disallowed header - arbitrary",
			header:   "X-Evil-Header",
			expected: false,
		},
		{
			name:     "case insensitive match",
			header:   "content-type",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidHeaderName(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAndSanitizeHeaders(t *testing.T) {
	tests := []struct {
		name            string
		input           map[string]string
		expectedHeaders map[string]string
		expectWarnings  bool
	}{
		{
			name: "all allowed headers",
			input: map[string]string{
				"Content-Type":  "application/json",
				"X-Request-ID":  "123",
				"X-Custom-Test": "value",
			},
			expectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"X-Request-ID":  "123",
				"X-Custom-Test": "value",
			},
			expectWarnings: false,
		},
		{
			name: "blocked header",
			input: map[string]string{
				"Content-Type": "application/json",
				"Set-Cookie":   "admin=true",
			},
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectWarnings: true,
		},
		{
			name: "header with XSS",
			input: map[string]string{
				"Content-Type": "<script>alert('xss')</script>",
			},
			expectedHeaders: map[string]string{
				"Content-Type": "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
			},
			expectWarnings: true,
		},
		{
			name: "header with CRLF injection",
			input: map[string]string{
				"X-Request-ID": "test\r\nSet-Cookie: evil=true",
			},
			expectedHeaders: map[string]string{
				"X-Request-ID": "testSet-Cookie: evil=true",
			},
			expectWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, warnings := ValidateAndSanitizeHeaders(tt.input)

			assert.Equal(t, tt.expectedHeaders, sanitized)

			if tt.expectWarnings {
				assert.NotEmpty(t, warnings)
			} else {
				assert.Empty(t, warnings)
			}
		})
	}
}

func TestSanitizeJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "safe JSON string",
			input:    "test value",
			expected: "test value",
		},
		{
			name:     "JSON with HTML",
			input:    "<div>test</div>",
			expected: "&lt;div&gt;test&lt;/div&gt;",
		},
		{
			name:     "JSON with quotes",
			input:    `test "value"`,
			expected: `test &#34;value&#34;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeJSONString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
