package httpx

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

// compileInputValidator compiles one typed validation function so request path does not
// repeat reflection shape checks for every invocation.
func compileInputValidator[I any](v *validator.Validate) func(*I) error {
	if v == nil {
		return nil
	}

	inputType := reflect.TypeFor[I]()
	hasNestedPointer := false
	for inputType.Kind() == reflect.Pointer {
		hasNestedPointer = true
		inputType = inputType.Elem()
	}
	if inputType.Kind() != reflect.Struct {
		return nil
	}

	if !hasNestedPointer {
		return func(input *I) error {
			if input == nil {
				return nil
			}
			return v.Struct(input)
		}
	}

	return func(input *I) error {
		if input == nil {
			return nil
		}

		value := reflect.ValueOf(input)
		for value.IsValid() && value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return nil
			}
			value = value.Elem()
		}
		if !value.IsValid() || value.Kind() != reflect.Struct {
			return nil
		}

		return v.Struct(input)
	}
}

// validationErrorMessage converts validator errors into a concise HTTP-facing message.
func validationErrorMessage(err error) string {
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return "request validation failed"
	}

	issues := lo.Map(validationErrs, func(validationErr validator.FieldError, _ int) string {
		field := validationErr.Field()
		if field == "" {
			field = validationErr.StructField()
		}
		if field == "" {
			field = "input"
		}

		return field + " failed '" + validationErr.Tag() + "'"
	})

	if len(issues) == 0 {
		return "request validation failed"
	}

	return "request validation failed: " + strings.Join(issues, "; ")
}
