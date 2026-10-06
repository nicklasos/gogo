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
	"time"

	"app/config"
	"app/internal"
	"app/internal/cache"
	"app/internal/db"
	"app/internal/logger"
	"app/internal/mail"
	custommiddleware "app/internal/middleware"
	"app/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

const TestJWTSecret = "test-secret-key"

// TestServer wraps httptest.Server with helper methods
type TestServer struct {
	server *httptest.Server
	router *gin.Engine

	// Mail collects every message the application sent during the test
	Mail *mail.MemorySender
}

// TestResponse represents an HTTP response for testing
type TestResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

// GenerateTestJWT creates a valid JWT token for testing
func GenerateTestJWT(userID int32, email string) string {
	claims := &custommiddleware.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(TestJWTSecret))
	if err != nil {
		panic(fmt.Sprintf("Failed to sign test JWT: %v", err))
	}
	return tokenString
}

// CreateTestServer creates a test server with transaction-scoped database queries.
// Pass configure functions to change the config before the routes are registered.
func CreateTestServer(t *testing.T, ctx context.Context, tx pgx.Tx, queries *db.Queries, configure ...func(*config.Config)) *TestServer {
	gin.SetMode(gin.TestMode)

	testLogger, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	testConfig := &config.Config{
		UploadFolder: t.TempDir(),
		FilesBaseURL: "http://localhost:8181/api/files",
		JWTSecret:    TestJWTSecret,
		AppName:      "TestApp",
		FrontendURL:  "http://localhost:5173",

		AllowRegistration: true,
		AuthRateLimit:     true,
		TrustedProxies:    []string{"127.0.0.1", "::1"},
	}

	for _, fn := range configure {
		fn(testConfig)
	}

	router := server.NewEngine(testConfig, testLogger, nil)
	testMail := mail.NewMemorySender()

	app := &internal.App{
		Config:  testConfig,
		Queries: queries,
		Tx:      db.NewTxRunner(tx, queries),
		Cache:   cache.NewMemoryCache(),
		Logger:  testLogger,
		Mail:    testMail,
		Api:     router.Group("/api/v1"),
	}

	server.RegisterRoutes(router, app)

	return &TestServer{
		server: httptest.NewServer(router),
		router: router,
		Mail:   testMail,
	}
}

// Close closes the test server
func (ts *TestServer) Close() {
	ts.server.Close()
}

// GET makes a GET request to the test server
func (ts *TestServer) GET(path string) *TestResponse {
	return ts.makeRequest("GET", path, nil, nil)
}

// GETAuth makes an authenticated GET request
func (ts *TestServer) GETAuth(path, token string) *TestResponse {
	return ts.makeRequest("GET", path, nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

// POST makes a POST request to the test server
func (ts *TestServer) POST(path string, body interface{}) *TestResponse {
	return ts.makeRequest("POST", path, body, nil)
}

// POSTAuth makes an authenticated POST request
func (ts *TestServer) POSTAuth(path string, body interface{}, token string) *TestResponse {
	return ts.makeRequest("POST", path, body, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

// PUT makes a PUT request to the test server
func (ts *TestServer) PUT(path string, body interface{}) *TestResponse {
	return ts.makeRequest("PUT", path, body, nil)
}

// PUTAuth makes an authenticated PUT request
func (ts *TestServer) PUTAuth(path string, body interface{}, token string) *TestResponse {
	return ts.makeRequest("PUT", path, body, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

// DELETE makes a DELETE request to the test server
func (ts *TestServer) DELETE(path string) *TestResponse {
	return ts.makeRequest("DELETE", path, nil, nil)
}

// DELETEAuth makes an authenticated DELETE request
func (ts *TestServer) DELETEAuth(path, token string) *TestResponse {
	return ts.makeRequest("DELETE", path, nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

func (ts *TestServer) makeRequest(method, path string, body interface{}, headers map[string]string) *TestResponse {
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

	url := ts.server.URL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		panic(fmt.Sprintf("Failed to create request: %v", err))
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
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
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
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
