package errs

// Common error keys
const (
	ErrKeyUnauthorized  = "unauthorized"
	ErrKeyForbidden     = "forbidden"
	ErrKeyNotFound      = "not_found"
	ErrKeyBadRequest    = "bad_request"
	ErrKeyInternalError = "internal_error"
	ErrKeyInvalidFormat = "invalid_format"
)

// Auth error keys
const (
	ErrKeyAuthInvalidToken       = "auth.invalid_token"
	ErrKeyAuthUserNotFound       = "auth.user_not_found"
	ErrKeyAuthInvalidCredentials = "auth.invalid_credentials"
	ErrKeyAuthTokenRequired      = "auth.token_required"
	ErrKeyAuthUserExists         = "auth.user_exists"
)

// Example error keys
const (
	ErrKeyExampleNotFound  = "examples.not_found"
	ErrKeyExampleInvalidID = "examples.invalid_id"
)

// Validation error keys
const (
	ErrKeyValidationFailed       = "validation.failed"
	ErrKeyValidationRequired     = "validation.required"
	ErrKeyValidationEmail        = "validation.email"
	ErrKeyValidationMin          = "validation.min"
	ErrKeyValidationMax          = "validation.max"
	ErrKeyValidationOneOf        = "validation.oneof"
	ErrKeyValidationNumeric      = "validation.numeric"
	ErrKeyValidationAlpha        = "validation.alpha"
	ErrKeyValidationAlphanum     = "validation.alphanum"
	ErrKeyValidationURL          = "validation.url"
	ErrKeyValidationUUID         = "validation.uuid"
	ErrKeyValidationInvalid       = "validation.invalid"
	ErrKeyValidationBodyInvalid  = "validation.body_invalid"
	ErrKeyValidationTypeMismatch = "validation.type_mismatch"
)

// GetValidationErrorKey returns the error key for a validation rule
func GetValidationErrorKey(rule string) string {
	switch rule {
	case "required":
		return ErrKeyValidationRequired
	case "email":
		return ErrKeyValidationEmail
	case "min":
		return ErrKeyValidationMin
	case "max":
		return ErrKeyValidationMax
	case "oneof":
		return ErrKeyValidationOneOf
	case "numeric":
		return ErrKeyValidationNumeric
	case "alpha":
		return ErrKeyValidationAlpha
	case "alphanum":
		return ErrKeyValidationAlphanum
	case "url":
		return ErrKeyValidationURL
	case "uuid":
		return ErrKeyValidationUUID
	default:
		return ErrKeyValidationInvalid
	}
}

// GetFieldValidationErrorKey returns a field-specific validation error key
// Format: validation.{field}.{rule}
func GetFieldValidationErrorKey(field, rule string) string {
	baseKey := GetValidationErrorKey(rule)
	// Convert validation.required to validation.{field}.required
	if baseKey == ErrKeyValidationRequired {
		return "validation." + field + ".required"
	}
	if baseKey == ErrKeyValidationEmail {
		return "validation." + field + ".email"
	}
	if baseKey == ErrKeyValidationMin {
		return "validation." + field + ".min"
	}
	if baseKey == ErrKeyValidationMax {
		return "validation." + field + ".max"
	}
	if baseKey == ErrKeyValidationOneOf {
		return "validation." + field + ".oneof"
	}
	if baseKey == ErrKeyValidationNumeric {
		return "validation." + field + ".numeric"
	}
	if baseKey == ErrKeyValidationAlpha {
		return "validation." + field + ".alpha"
	}
	if baseKey == ErrKeyValidationAlphanum {
		return "validation." + field + ".alphanum"
	}
	if baseKey == ErrKeyValidationURL {
		return "validation." + field + ".url"
	}
	if baseKey == ErrKeyValidationUUID {
		return "validation." + field + ".uuid"
	}
	return "validation." + field + ".invalid"
}
