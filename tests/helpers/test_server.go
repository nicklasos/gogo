package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal"
	"app/internal/auth"
	"app/internal/db"
	"app/internal/example"
	"app/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TestServer wraps httptest.Server with helper methods
type TestServer struct {
	server *httptest.Server
	router *gin.Engine
}

// TestResponse represents an HTTP response for testing
type TestResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

// CreateTestServer creates a test server with transaction-scoped database queries
func CreateTestServer(t *testing.T, ctx context.Context, tx pgx.Tx, queries *db.Queries) *TestServer {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router
	router := gin.New()

	// Logger
	testLogger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout", // or a writer that captures logs for tests
	})
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	// Create minimal app structure for testing
	app := &internal.App{
		Queries: queries,
		Logger:  testLogger,
		Api:     router.Group("/api/v1"),
	}

	// Register auth routes
	jwtSecret := []byte("test-secret-key")
	authService := auth.NewAuthService(queries, jwtSecret, testLogger)
	authHandler := auth.NewAuthHandler(authService, testLogger)
	auth.RegisterRoutes(app.Api, authHandler, authService)

	// Register example routes
	example.RegisterRoutes(app, authService)

	// Create test server
	server := httptest.NewServer(router)

	return &TestServer{
		server: server,
		router: router,
	}
}

// Close closes the test server
func (ts *TestServer) Close() {
	ts.server.Close()
}

// GET makes a GET request to the test server
func (ts *TestServer) GET(path string) *TestResponse {
	return ts.makeRequest("GET", path, nil)
}

// POST makes a POST request to the test server
func (ts *TestServer) POST(path string, body interface{}) *TestResponse {
	var bodyReader io.Reader
	if body != nil {
		if str, ok := body.(string); ok {
			bodyReader = strings.NewReader(str)
		} else if reader, ok := body.(io.Reader); ok {
			bodyReader = reader
		} else {
			bodyBytes, _ := json.Marshal(body)
			bodyReader = strings.NewReader(string(bodyBytes))
		}
	}
	return ts.makeRequest("POST", path, bodyReader)
}

// PUT makes a PUT request to the test server
func (ts *TestServer) PUT(path string, body interface{}) *TestResponse {
	var bodyReader io.Reader
	if body != nil {
		if str, ok := body.(string); ok {
			bodyReader = strings.NewReader(str)
		} else if reader, ok := body.(io.Reader); ok {
			bodyReader = reader
		} else {
			bodyBytes, _ := json.Marshal(body)
			bodyReader = strings.NewReader(string(bodyBytes))
		}
	}
	return ts.makeRequest("PUT", path, bodyReader)
}

// DELETE makes a DELETE request to the test server
func (ts *TestServer) DELETE(path string) *TestResponse {
	return ts.makeRequest("DELETE", path, nil)
}

// makeRequest makes an HTTP request to the test server
func (ts *TestServer) makeRequest(method, path string, body io.Reader) *TestResponse {
	url := ts.server.URL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(fmt.Sprintf("Failed to create request: %v", err))
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to make request: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response body: %v", err))
	}

	return &TestResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Header:     resp.Header,
	}
}

// JSON unmarshals the response body as JSON
func (tr *TestResponse) JSON(v interface{}) error {
	return json.Unmarshal(tr.Body, v)
}

// String returns the response body as a string
func (tr *TestResponse) String() string {
	return string(tr.Body)
}

// NewRequest creates a new HTTP request for testing
func (ts *TestServer) NewRequest(method, path string, body io.Reader) *http.Request {
	url := ts.server.URL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(fmt.Sprintf("Failed to create request: %v", err))
	}
	return req
}

// Do executes an HTTP request and returns the response
func (ts *TestServer) Do(req *http.Request) *TestResponse {
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to make request: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response body: %v", err))
	}

	return &TestResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Header:     resp.Header,
	}
}

// StringToReadCloser converts a string to io.ReadCloser
func StringToReadCloser(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}
