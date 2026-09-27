package todo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrValidation marks input that is well-formed but breaks a business rule.
var ErrValidation = errors.New("validation error")

// Validator validates a value against the `validate` struct tags on its fields.
type Validator interface {
	Struct(s any) error
}

// NewValidator builds a standard go-playground validator instance.
func NewValidator() (*validator.Validate, error) {
	v := validator.New()
	// No custom rule registrations needed!
	return v, nil
}

// inputError wraps a validator failure in ErrValidation with a concise description.
func inputError(err error) error {
	if err == nil {
		return nil
	}

	var fieldErrors validator.ValidationErrors
	if errors.As(err, &fieldErrors) {
		return fmt.Errorf("%w: %s", ErrValidation, describe(fieldErrors))
	}
	return fmt.Errorf("%w: %s", ErrValidation, err)
}

// describe renders built-in validator violations as human-readable messages.
func describe(errs validator.ValidationErrors) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		field := strings.ToLower(e.Field())
		switch e.Tag() {
		case "required":
			parts = append(parts, field+" is required")
		case "max":
			parts = append(parts, fmt.Sprintf("%s must be at most %s characters", field, e.Param()))
		case "min":
			parts = append(parts, fmt.Sprintf("%s must be at least %s characters", field, e.Param()))
		case "oneof":
			parts = append(parts, fmt.Sprintf("%s must be one of [%s]", field, strings.ReplaceAll(e.Param(), " ", ", ")))
		default:
			parts = append(parts, fmt.Sprintf("%s is invalid", field))
		}
	}
	return strings.Join(parts, ", ")
}
