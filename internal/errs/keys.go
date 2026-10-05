package errs

// Common error keys
const (
	ErrKeyUnauthorized  = "unauthorized"
	ErrKeyForbidden     = "forbidden"
	ErrKeyNotFound      = "not_found"
	ErrKeyBadRequest    = "bad_request"
	ErrKeyInternalError = "internal_error"
)

// Auth error keys
const (
	ErrKeyAuthInvalidToken       = "auth.invalid_token"
	ErrKeyAuthUserNotFound       = "auth.user_not_found"
	ErrKeyAuthInvalidCredentials = "auth.invalid_credentials"
	ErrKeyAuthTokenRequired      = "auth.token_required"
	ErrKeyAuthUserExists         = "auth.user_exists"
	ErrKeyAuthInvalidCurrentPass = "auth.invalid_current_password"
	ErrKeyAuthTooManyRequests    = "auth.too_many_requests"
	ErrKeyAuthRegistrationClosed = "auth.registration_disabled"
	ErrKeyAuthInvalidEmailToken  = "auth.invalid_or_expired_link"
	ErrKeyAuthEmailVerified      = "auth.email_already_verified"
)

// Users error keys
const (
	ErrKeyUsersNotFound         = "users.not_found"
	ErrKeyUsersForbiddenRole    = "users.forbidden_role"
	ErrKeyUsersCannotDeleteSelf = "users.cannot_delete_self"
)

// Example error keys
const (
	ErrKeyExampleNotFound  = "examples.not_found"
	ErrKeyExampleInvalidID = "examples.invalid_id"
)

// Upload error keys
const (
	ErrKeyUploadNotFound       = "uploads.not_found"
	ErrKeyUploadTypeNotAllowed = "uploads.type_not_allowed"
	ErrKeyUploadTooLarge       = "uploads.too_large"
	ErrKeyUploadEmpty          = "uploads.empty"
)

// Validation error keys
const (
	ErrKeyValidationFailed       = "validation.failed"
	ErrKeyValidationInvalid      = "validation.invalid"
	ErrKeyValidationBodyInvalid  = "validation.body_invalid"
	ErrKeyValidationTypeMismatch = "validation.type_mismatch"
)

var validationRules = map[string]bool{
	"required": true, "email": true, "min": true, "max": true, "oneof": true,
	"numeric": true, "alpha": true, "alphanum": true, "url": true, "uuid": true,
}

// GetFieldValidationErrorKey returns the key for a failed rule on a field:
// validation.{field}.{rule}, or validation.{field}.invalid for a rule clients have no text for.
func GetFieldValidationErrorKey(field, rule string) string {
	if !validationRules[rule] {
		rule = "invalid"
	}
	return "validation." + field + "." + rule
}
