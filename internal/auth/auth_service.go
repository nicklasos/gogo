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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	queries   *db.Queries
	tx        *db.TxRunner
	jwtSecret []byte
	logger    *logger.Logger
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var (
	ErrInvalidCredentials = errs.NewUnauthorizedError(errs.ErrKeyAuthInvalidCredentials, "Invalid email or password")
	ErrUserNotFound       = errs.NewNotFoundError(errs.ErrKeyAuthUserNotFound, "User not found")
	ErrInvalidToken       = errs.NewUnauthorizedError(errs.ErrKeyAuthInvalidToken, "Invalid token")
	ErrTokenExpired       = errs.NewUnauthorizedError(errs.ErrKeyAuthInvalidToken, "Token expired")
	ErrUserAlreadyExists  = errs.NewBadRequestError(errs.ErrKeyAuthUserExists, "User with this email already exists")
	// 400 rather than 401: a 401 would make API clients try to refresh the session
	ErrInvalidCurrentPassword = errs.NewBadRequestError(errs.ErrKeyAuthInvalidCurrentPass, "Current password is incorrect")
)

func NewAuthService(queries *db.Queries, tx *db.TxRunner, jwtSecret []byte, logger *logger.Logger) *AuthService {
	return &AuthService{
		queries:   queries,
		tx:        tx,
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
		Roles:    []string{middleware.RoleUser},
	})
	if err != nil {
		if domainErr := errs.DomainErrorFromPostgresUniqueViolation(err); domainErr != nil {
			return nil, nil, ErrUserAlreadyExists
		}
		return nil, nil, errs.WrapDatabaseError(err)
	}

	// Generate token pair
	tokenPair, err := s.generateTokenPair(ctx, s.queries, user)
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
	tokenPair, err := s.generateTokenPair(ctx, s.queries, user)
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

	var tokenPair *TokenPair
	err = s.tx.WithTx(ctx, func(q *db.Queries) error {
		if err := q.RevokeRefreshToken(ctx, refreshToken); err != nil {
			return errs.WrapDatabaseError(err)
		}
		tokenPair, err = s.generateTokenPair(ctx, q, user)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	return tokenPair, nil
}

func (s *AuthService) generateTokenPair(ctx context.Context, queries *db.Queries, user db.User) (*TokenPair, error) {
	jtiBytes := make([]byte, 8)
	if _, err := rand.Read(jtiBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token id: %w", err)
	}

	// Generate access token (7 days)
	accessClaims := &middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        hex.EncodeToString(jtiBytes),
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
	_, err = queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
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

// GetUserRoles is called on every authenticated request. A token whose user no longer
// exists is an invalid token (401), so clients sign out instead of showing "not found".
func (s *AuthService) GetUserRoles(ctx context.Context, userID int32) ([]string, error) {
	user, err := s.queries.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}
	return user.Roles, nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID int32, email, name string) (*db.User, error) {
	user, err := s.queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:    userID,
		Email: email,
		Name:  name,
	})
	if err != nil {
		if domainErr := errs.DomainErrorFromPostgresUniqueViolation(err); domainErr != nil {
			return nil, ErrUserAlreadyExists
		}
		return nil, errs.WrapDatabaseError(err)
	}
	return &user, nil
}

// UpdatePassword changes the password and revokes every refresh token, so other sessions
// cannot be renewed with the old credentials.
func (s *AuthService) UpdatePassword(ctx context.Context, userID int32, currentPassword, newPassword string) error {
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return ErrInvalidCurrentPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.tx.WithTx(ctx, func(q *db.Queries) error {
		if err := q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
			ID:       userID,
			Password: string(hashedPassword),
		}); err != nil {
			return errs.WrapDatabaseError(err)
		}
		return errs.WrapDatabaseError(q.RevokeAllUserRefreshTokens(ctx, userID))
	})
}

// Logout revokes all refresh tokens for the user
func (s *AuthService) Logout(ctx context.Context, userID int32) error {
	if err := s.queries.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return errs.WrapDatabaseError(err)
	}
	return nil
}
