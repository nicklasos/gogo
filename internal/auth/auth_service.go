package auth

import (
	"app/internal/db"
	"app/internal/errs"
	"app/internal/logger"
	"app/internal/middleware"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	queries   *db.Queries
	jwtSecret []byte
	logger    *logger.Logger
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RegisterRequest represents the request structure for user registration
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest represents the request structure for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

var (
	ErrInvalidCredentials = errs.NewUnauthorizedError(errs.ErrKeyAuthInvalidCredentials, "Invalid email or password")
	ErrUserNotFound       = errs.NewNotFoundError(errs.ErrKeyAuthUserNotFound, "User not found")
	ErrInvalidToken       = errs.NewUnauthorizedError(errs.ErrKeyAuthInvalidToken, "Invalid token")
	ErrTokenExpired       = errs.NewUnauthorizedError(errs.ErrKeyAuthInvalidToken, "Token expired")
	ErrUserAlreadyExists  = errs.NewBadRequestError(errs.ErrKeyAuthUserExists, "User with this email already exists")
)

func NewAuthService(queries *db.Queries, jwtSecret []byte, logger *logger.Logger) *AuthService {
	return &AuthService{
		queries:   queries,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*TokenPair, *db.User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user (let DB enforce uniqueness to avoid race conditions)
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashedPassword),
	})
	if err != nil {
		// Map unique violations to a stable error
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, nil, ErrUserAlreadyExists
		}
		// Fallback for driver/driver-text wrapped errors
		msg := err.Error()
		if strings.Contains(msg, "SQLSTATE 23505") || strings.Contains(msg, "duplicate key value") {
			return nil, nil, ErrUserAlreadyExists
		}
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate token pair
	tokenPair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, &user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*TokenPair, *db.User, error) {
	// Get user by email
	user, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Generate token pair
	tokenPair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, &user, nil
}

// RefreshToken generates a new token pair using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Get refresh token from database
	dbToken, err := s.queries.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Get user
	user, err := s.queries.GetUserByID(ctx, dbToken.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Revoke old refresh token
	err = s.queries.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		// Log error but continue
	}

	// Generate new token pair
	tokenPair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, nil
}

func (s *AuthService) generateTokenPair(ctx context.Context, user db.User) (*TokenPair, error) {
	// Generate access token (7 days)
	accessClaims := &middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token (30 days)
	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenString := hex.EncodeToString(refreshTokenBytes)

	// Store refresh token in database
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	_, err = s.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     refreshTokenString,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

func (s *AuthService) VerifyJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return token, nil
}

func (s *AuthService) GetUserFromContext(ctx context.Context, userID int32) (*db.User, error) {
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}
