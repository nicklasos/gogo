package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"app/internal/db"
	"app/internal/logger"
	"app/internal/middleware"
)

// GetTestLogger creates a test logger
func GetTestLogger(t *testing.T) *logger.Logger {
	testLogger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create test logger")
	return testLogger
}

// TestJPEG is a minimal payload that is recognised as a JPEG image by content sniffing.
var TestJPEG = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, []byte("test image content")...)

// TestPDF is a minimal payload that is recognised as a PDF document.
var TestPDF = []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\ntrailer\n<<>>\n%%EOF\n")

// Actor is the user as a policy sees them.
func Actor(user *db.User) middleware.Actor {
	return middleware.Actor{ID: user.ID, Roles: user.Roles}
}
