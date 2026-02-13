package validator

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Name        string `validate:"required,no_spaces"`
	Description string `validate:"required"`
	Email       string `validate:"email"`
	URL         string `validate:"url"`
}

func TestCustomValidator_Validate(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		input     interface{}
		wantError bool
		errMsg    string
	}{
		{
			name: "valid struct",
			input: &TestStruct{
				Name:        "test-name",
				Description: "Test description",
				Email:       "test@example.com",
				URL:         "https://example.com",
			},
			wantError: false,
		},
		{
			name: "missing required field",
			input: &TestStruct{
				Name:  "test-name",
				Email: "test@example.com",
				URL:   "https://example.com",
				// Description missing
			},
			wantError: true,
			errMsg:    "Description is required",
		},
		{
			name: "name with spaces",
			input: &TestStruct{
				Name:        "test name", // has space
				Description: "Test description",
				Email:       "test@example.com",
				URL:         "https://example.com",
			},
			wantError: true,
			errMsg:    "Name cannot contain spaces",
		},
		{
			name: "invalid email",
			input: &TestStruct{
				Name:        "test-name",
				Description: "Test description",
				Email:       "not-an-email", // invalid email
				URL:         "https://example.com",
			},
			wantError: true,
			errMsg:    "Email must be a valid email address",
		},
		{
			name: "invalid URL",
			input: &TestStruct{
				Name:        "test-name",
				Description: "Test description",
				Email:       "test@example.com",
				URL:         "not-a-url", // invalid URL
			},
			wantError: true,
			errMsg:    "URL must be a valid URL",
		},
		{
			name: "multiple validation errors",
			input: &TestStruct{
				Name:  "test name", // has space
				Email: "test@example.com",
				URL:   "https://example.com",
				// Description missing
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errMsg != "" {
					httpErr, ok := err.(*echo.HTTPError)
					assert.True(t, ok, "error should be *echo.HTTPError")
					assert.Equal(t, http.StatusBadRequest, httpErr.Code)
					assert.Contains(t, httpErr.Message, tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNoSpaces(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "no spaces",
			value:     "test-name-123",
			wantError: false,
		},
		{
			name:      "with spaces",
			value:     "test name",
			wantError: true,
		},
		{
			name:      "empty string",
			value:     "",
			wantError: false, // no_spaces only checks for spaces, not emptiness
		},
		{
			name:      "special characters",
			value:     "test_name-123",
			wantError: false,
		},
		{
			name:      "multiple spaces",
			value:     "test   name",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestInput struct {
				Value string `validate:"no_spaces"`
			}

			input := &TestInput{Value: tt.value}
			err := v.Validate(input)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatValidationErrors(t *testing.T) {
	v := NewValidator()

	type ComplexStruct struct {
		Name  string `validate:"required,no_spaces,min=3,max=50"`
		Email string `validate:"required,email"`
		Age   int    `validate:"min=0,max=150"`
	}

	tests := []struct {
		name           string
		input          *ComplexStruct
		expectedInMsg  string
	}{
		{
			name: "min length violation",
			input: &ComplexStruct{
				Name:  "ab", // too short
				Email: "test@example.com",
				Age:   25,
			},
			expectedInMsg: "must be at least",
		},
		{
			name: "required field missing",
			input: &ComplexStruct{
				Name: "valid-name",
				// Email missing
				Age: 25,
			},
			expectedInMsg: "Email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.input)
			assert.Error(t, err)

			httpErr, ok := err.(*echo.HTTPError)
			assert.True(t, ok)
			assert.Contains(t, httpErr.Message, tt.expectedInMsg)
		})
	}
}
