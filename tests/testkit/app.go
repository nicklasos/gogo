// Package testkit is what a test starts from:
//
//	func TestExamples_List(t *testing.T) {
//		app := testkit.New(t)
//		user := app.Create.User()
//		app.Create.Example(user.ID, factory.WithTitle("Hello"))
//
//		app.As(user).GET("/api/v1/examples").
//			AssertOK().
//			AssertJSONPath("data.0.title", "Hello")
//	}
//
// New opens a transaction that is rolled back when the test ends, so tests never see each
// other's rows. Everything hangs off the returned App: factories (app.Create), requests
// (app.GET, app.As(user).POST, ...), database assertions and the fakes (app.Mail).
package testkit

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"app/config"
	"app/internal"
	"app/internal/cache"
	"app/internal/db"
	"app/internal/factory"
	"app/internal/logger"
	"app/internal/mail"
	"app/internal/middleware"
	"app/internal/server"
	"app/tests"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	pulse "github.com/nicklasos/gopulse"
)

const JWTSecret = "test-secret-key"

type App struct {
	// Client sends requests without a token. Use As for an authenticated one.
	*Client

	Ctx     context.Context
	Tx      pgx.Tx
	Queries *db.Queries
	Create  *factory.Factory

	// TxRunner is for a service that opens transactions: inside the test they are savepoints
	TxRunner *db.TxRunner

	Config *config.Config
	Logger *logger.Logger
	Cache  *cache.MemoryCache

	// Mail collects every message the application sent during the test
	Mail *mail.MemorySender

	// Pulse turns monitoring on when it is set before the first request
	Pulse *pulse.Pulse

	t      *testing.T
	server *httptest.Server
}

// New starts a test. Pass configure functions to change the config:
//
//	app := testkit.New(t, func(cfg *config.Config) { cfg.AllowRegistration = false })
func New(t *testing.T, configure ...func(*config.Config)) *App {
	t.Helper()

	ctx := context.Background()
	tx, err := tests.GetTestDBPool().Begin(ctx)
	if err != nil {
		t.Fatalf("testkit: begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	if err != nil {
		t.Fatalf("testkit: create logger: %v", err)
	}

	cfg := &config.Config{
		UploadFolder: t.TempDir(),
		FilesBaseURL: "http://localhost:8181/api/files",
		JWTSecret:    JWTSecret,
		AppName:      "TestApp",
		FrontendURL:  "http://localhost:5173",

		AllowRegistration: true,
		AuthRateLimit:     true,
		TrustedProxies:    []string{"127.0.0.1", "::1"},
	}
	for _, fn := range configure {
		fn(cfg)
	}

	queries := db.New(tx)
	app := &App{
		Ctx:      ctx,
		Tx:       tx,
		Queries:  queries,
		Create:   factory.New(t, tx),
		TxRunner: db.NewTxRunner(tx, queries),
		Config:   cfg,
		Logger:   log,
		Cache:    cache.NewMemoryCache(),
		Mail:     mail.NewMemorySender(),
		t:        t,
	}
	app.Client = &Client{app: app}
	return app
}

// As returns a client that sends every request as the given user.
func (a *App) As(user *db.User) *Client {
	return a.WithToken(a.Token(user))
}

// WithToken returns a client that sends the access token, for one the application issued itself.
func (a *App) WithToken(token string) *Client {
	return a.WithHeader("Authorization", "Bearer "+token)
}

// Token is a valid access token for the user.
func (a *App) Token(user *db.User) string {
	a.t.Helper()

	claims := &middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(a.Config.JWTSecret))
	if err != nil {
		a.t.Fatalf("testkit: sign token: %v", err)
	}
	return token
}

// URL is the address of the test server for a path, for the rare request the client cannot make.
func (a *App) URL(path string) string {
	return a.baseURL() + path
}

// baseURL starts the server on first use, so a test that makes no requests never builds one.
// It runs the same engine and routes as production, on top of the test transaction.
func (a *App) baseURL() string {
	if a.server != nil {
		return a.server.URL
	}

	gin.SetMode(gin.TestMode)
	router := server.NewEngine(a.Config, a.Logger, a.Pulse)
	server.RegisterRoutes(router, &internal.App{
		Config:  a.Config,
		Queries: a.Queries,
		Tx:      a.TxRunner,
		Cache:   a.Cache,
		Logger:  a.Logger,
		Mail:    a.Mail,
		Pulse:   a.Pulse,
		Api:     router.Group("/api/v1"),
	})

	a.server = httptest.NewServer(router)
	a.t.Cleanup(a.server.Close)
	return a.server.URL
}

// Actor is the user as a policy sees them, for calling a service directly.
func Actor(user *db.User) middleware.Actor {
	return middleware.Actor{ID: user.ID, Roles: user.Roles}
}

// JPEG is a minimal payload that is recognised as a JPEG image by content sniffing.
var JPEG = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, []byte("test image content")...)

// PDF is a minimal payload that is recognised as a PDF document.
var PDF = []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\ntrailer\n<<>>\n%%EOF\n")
