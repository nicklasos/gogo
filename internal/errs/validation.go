package errs

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	// Configure validator to use JSON field names instead of struct field names
	// This ensures fieldError.Field() returns JSON names (e.g., "email") instead of struct names (e.g., "Email")
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}
}

type ValidationErrorResponse struct {
	Message  string              `json:"message"`
	ErrorKey string              `json:"error_key"`
	Errors   map[string][]string `json:"errors"`
}

// FormatValidationError formats validation errors into a Laravel-style response with error keys
func FormatValidationError(err error) ValidationErrorResponse {
	validationErrors := make(map[string][]string)
	errorMessage := "The given data was invalid."

	if err == nil {
		return ValidationErrorResponse{
			Message:  errorMessage,
			ErrorKey: ErrKeyValidationFailed,
			Errors:   validationErrors,
		}
	}

	var validatorErrors validator.ValidationErrors
	if errors.As(err, &validatorErrors) {
		for _, fieldError := range validatorErrors {
			// Field() now returns JSON field name thanks to RegisterTagNameFunc
			fieldName := fieldError.Field()
			// Get error key instead of message
			errorKey := GetFieldValidationErrorKey(fieldName, fieldError.Tag())

			if _, exists := validationErrors[fieldName]; !exists {
				validationErrors[fieldName] = []string{}
			}
			validationErrors[fieldName] = append(validationErrors[fieldName], errorKey)
		}
	} else {
		handleNonValidationError(err, validationErrors)
	}

	return ValidationErrorResponse{
		Message:  errorMessage,
		ErrorKey: ErrKeyValidationFailed,
		Errors:   validationErrors,
	}
}

// getUserFriendlyMessage creates user-friendly error messages from validator.FieldError
func getUserFriendlyMessage(fieldError validator.FieldError) string {
	fieldName := formatFieldNameForDisplay(fieldError.Field())
	tag := fieldError.Tag()
	param := fieldError.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("The %s field is required.", fieldName)
	case "email":
		return fmt.Sprintf("The %s must be a valid email address.", fieldName)
	case "min":
		if fieldError.Type().Kind().String() == "string" {
			return fmt.Sprintf("The %s must be at least %s characters.", fieldName, param)
		}
		return fmt.Sprintf("The %s must be at least %s.", fieldName, param)
	case "max":
		if fieldError.Type().Kind().String() == "string" {
			return fmt.Sprintf("The %s may not be greater than %s characters.", fieldName, param)
		}
		return fmt.Sprintf("The %s may not be greater than %s.", fieldName, param)
	case "oneof":
		return fmt.Sprintf("The %s must be one of: %s.", fieldName, strings.ReplaceAll(param, " ", ", "))
	case "numeric":
		return fmt.Sprintf("The %s must be a number.", fieldName)
	case "alpha":
		return fmt.Sprintf("The %s may only contain letters.", fieldName)
	case "alphanum":
		return fmt.Sprintf("The %s may only contain letters and numbers.", fieldName)
	case "url":
		return fmt.Sprintf("The %s must be a valid URL.", fieldName)
	case "uuid":
		return fmt.Sprintf("The %s must be a valid UUID.", fieldName)
	default:
		return fmt.Sprintf("The %s field is invalid.", fieldName)
	}
}

// formatFieldNameForDisplay formats field names for user-facing messages
func formatFieldNameForDisplay(fieldName string) string {
	return strings.ReplaceAll(strings.ToLower(fieldName), "_", " ")
}

// handleNonValidationError handles errors that are not validator.ValidationErrors
// These are typically JSON unmarshal errors or malformed request body errors
func handleNonValidationError(err error, validationErrors map[string][]string) {
	errMsg := err.Error()

	fieldName := extractFieldFromJSONError(errMsg)
	if fieldName != "" {
		// Map JSON unmarshal errors to field-specific validation error keys
		baseKey := getJSONErrorKey(errMsg)
		// Create field-specific key: validation.{field}.type_mismatch
		fieldKey := "validation." + fieldName + "." + strings.TrimPrefix(baseKey, "validation.")
		validationErrors[fieldName] = []string{fieldKey}
		return
	}

	if strings.Contains(errMsg, "json:") || strings.Contains(errMsg, "EOF") || strings.Contains(errMsg, "cannot unmarshal") {
		validationErrors["body"] = []string{ErrKeyValidationBodyInvalid}
		return
	}

	validationErrors["general"] = []string{ErrKeyValidationInvalid}
}

// getJSONErrorKey maps JSON unmarshal errors to validation error keys
func getJSONErrorKey(errMsg string) string {
	if strings.Contains(errMsg, "cannot unmarshal") {
		if strings.Contains(errMsg, "of type int32") || strings.Contains(errMsg, "of type int64") {
			return ErrKeyValidationTypeMismatch
		}
		if strings.Contains(errMsg, "of type string") {
			return ErrKeyValidationTypeMismatch
		}
		if strings.Contains(errMsg, "of type bool") {
			return ErrKeyValidationTypeMismatch
		}
		return ErrKeyValidationTypeMismatch
	}

	if strings.Contains(errMsg, "invalid character") {
		return ErrKeyValidationBodyInvalid
	}

	if strings.Contains(errMsg, "EOF") {
		return ErrKeyValidationBodyInvalid
	}

	return ErrKeyValidationInvalid
}

// extractFieldFromJSONError extracts the field name from JSON unmarshal errors
// Examples:
//   - "json: cannot unmarshal number 1761901442695 into Go struct field CreateAssessmentInput.Questions.id of type int32"
//   - "json: cannot unmarshal string into Go struct field CreateAssessmentInput.title of type string"
func extractFieldFromJSONError(errMsg string) string {
	prefix := "Go struct field "
	idx := strings.Index(errMsg, prefix)
	if idx == -1 {
		return ""
	}

	rest := errMsg[idx+len(prefix):]
	ofTypeIdx := strings.Index(rest, " of type")
	if ofTypeIdx == -1 {
		return ""
	}

	fieldPath := strings.TrimSpace(rest[:ofTypeIdx])
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 {
		return ""
	}

	result := []string{}
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			result = append(result, strings.ToLower(parts[i]))
		}
	}

	return strings.Join(result, ".")
}

// extractJSONErrorMessage extracts a user-friendly error message from JSON errors
func extractJSONErrorMessage(errMsg string) string {
	if strings.Contains(errMsg, "cannot unmarshal") {
		if strings.Contains(errMsg, "into Go struct field") {
			if strings.Contains(errMsg, "of type int32") {
				return "The value is too large for this field."
			}
			if strings.Contains(errMsg, "of type int64") {
				return "The value must be a valid number."
			}
			if strings.Contains(errMsg, "of type string") {
				return "The value must be a string."
			}
			if strings.Contains(errMsg, "of type bool") {
				return "The value must be true or false."
			}
			return "The value format is invalid."
		}
		return "Invalid value format."
	}

	if strings.Contains(errMsg, "invalid character") {
		return "The request contains invalid characters."
	}

	if strings.Contains(errMsg, "EOF") {
		return "The request body is incomplete."
	}

	return "The request body is invalid or malformed."
}

// RespondWithValidationError sends a validation error response with error keys
func RespondWithValidationError(c *gin.Context, err error) {
	validationError := FormatValidationError(err)
	c.JSON(http.StatusBadRequest, validationError)
}
