package validator

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// CustomValidator wraps the validator instance
type CustomValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new validator with custom rules
func NewValidator() *CustomValidator {
	v := validator.New()

	// Register custom validation function for "no_spaces"
	_ = v.RegisterValidation("no_spaces", noSpaces)

	return &CustomValidator{
		validator: v,
	}
}

// Validate validates a struct based on its validation tags
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		// Type assert to ValidationErrors to get detailed error info
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				formatValidationErrors(validationErrors),
			)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

// noSpaces validates that a string doesn't contain spaces
func noSpaces(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return !strings.Contains(value, " ")
}

// formatValidationErrors formats validation errors into a user-friendly message
func formatValidationErrors(errs validator.ValidationErrors) string {
	var messages []string

	for _, err := range errs {
		var msg string
		field := err.Field()

		switch err.Tag() {
		case "required":
			msg = fmt.Sprintf("%s is required", field)
		case "no_spaces":
			msg = fmt.Sprintf("%s cannot contain spaces", field)
		case "email":
			msg = fmt.Sprintf("%s must be a valid email address", field)
		case "min":
			msg = fmt.Sprintf("%s must be at least %s characters", field, err.Param())
		case "max":
			msg = fmt.Sprintf("%s must be at most %s characters", field, err.Param())
		case "url":
			msg = fmt.Sprintf("%s must be a valid URL", field)
		default:
			msg = fmt.Sprintf("%s failed validation: %s", field, err.Tag())
		}

		messages = append(messages, msg)
	}

	return strings.Join(messages, "; ")
}
