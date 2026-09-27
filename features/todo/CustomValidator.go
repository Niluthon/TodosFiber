package todo

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrValidation marks input that is well-formed but breaks a business rule.
var ErrValidation = errors.New("validation error")

// Validator validates a value against the `validate` struct tags on its fields.
type Validator interface {
	Struct(s any) error
}

// NewValidator builds a standard go-playground validator instance with the
// custom `notblank` rule registered. `notblank` rejects values that are empty
// or contain only whitespace, which `required` alone does not catch.
func NewValidator() (*validator.Validate, error) {
	v := validator.New()
	if err := v.RegisterValidation("notblank", notBlank); err != nil {
		return nil, fmt.Errorf("register notblank validation: %w", err)
	}
	return v, nil
}

// notBlank reports whether the field holds a non-whitespace string. Pointer
// fields are dereferenced by the validator before this runs, but nil pointers
// are treated as valid so `omitempty` remains the source of truth for optional
// fields.
func notBlank(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return true
		}
		field = field.Elem()
	}
	if field.Kind() != reflect.String {
		return true
	}
	return strings.TrimSpace(field.String()) != ""
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
		case "notblank":
			parts = append(parts, field+" must not be blank")
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
