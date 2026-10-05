package errs

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	// Field errors are keyed by JSON name ("email"), which is what clients know, not by struct field ("Email")
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
			fieldName := fieldError.Field()
			validationErrors[fieldName] = append(validationErrors[fieldName], GetFieldValidationErrorKey(fieldName, fieldError.Tag()))
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

// handleNonValidationError handles errors that are not validator.ValidationErrors
// These are typically JSON unmarshal errors or malformed request body errors
func handleNonValidationError(err error, validationErrors map[string][]string) {
	errMsg := err.Error()

	fieldName := extractFieldFromJSONError(errMsg)
	if fieldName != "" {
		baseKey := getJSONErrorKey(errMsg)
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
	switch {
	case strings.Contains(errMsg, "cannot unmarshal"):
		return ErrKeyValidationTypeMismatch
	case strings.Contains(errMsg, "invalid character"), strings.Contains(errMsg, "EOF"):
		return ErrKeyValidationBodyInvalid
	default:
		return ErrKeyValidationInvalid
	}
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

// RespondWithValidationError sends a validation error response with error keys
func RespondWithValidationError(c *gin.Context, err error) {
	validationError := FormatValidationError(err)
	c.JSON(http.StatusBadRequest, validationError)
}
