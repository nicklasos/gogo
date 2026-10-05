package auth

import (
	"app/internal/db"
	"app/internal/errs"
	"app/internal/logger"
	"app/internal/mail"
	"app/internal/middleware"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

const (
	purposePasswordReset     = "password_reset"
	purposeEmailVerification = "email_verification"
)

// Options are the settings of the auth module. Zero durations and names get defaults.
type Options struct {
	JWTSecret            []byte
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	PasswordResetTTL     time.Duration
	EmailVerificationTTL time.Duration
	// FrontendURL is the base of the links sent in emails
	FrontendURL string
	AppName     string
}

func (o Options) withDefaults() Options {
	if o.AccessTokenTTL <= 0 {
		o.AccessTokenTTL = time.Hour
	}
	if o.RefreshTokenTTL <= 0 {
		o.RefreshTokenTTL = 30 * 24 * time.Hour
	}
	if o.PasswordResetTTL <= 0 {
		o.PasswordResetTTL = time.Hour
	}
	if o.EmailVerificationTTL <= 0 {
		o.EmailVerificationTTL = 48 * time.Hour
	}
	if o.AppName == "" {
		o.AppName = "App"
	}
	return o
}

type AuthService struct {
	queries *db.Queries
	tx      *db.TxRunner
	mail    mail.Sender
	logger  *logger.Logger
	opts    Options
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
	ErrInvalidEmailToken      = errs.NewBadRequestError(errs.ErrKeyAuthInvalidEmailToken, "This link is invalid or has expired")
	ErrEmailAlreadyVerified   = errs.NewBadRequestError(errs.ErrKeyAuthEmailVerified, "Email is already verified")
)

func NewAuthService(queries *db.Queries, tx *db.TxRunner, mailer mail.Sender, logger *logger.Logger, opts Options) *AuthService {
	return &AuthService{
		queries: queries,
		tx:      tx,
		mail:    mailer,
		logger:  logger,
		opts:    opts.withDefaults(),
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
	if err == nil {
		s.sendEmailVerification(ctx, user)
	}
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

	accessClaims := &middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        hex.EncodeToString(jtiBytes),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.opts.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.opts.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenString := hex.EncodeToString(refreshTokenBytes)

	// Store refresh token in database
	expiresAt := time.Now().Add(s.opts.RefreshTokenTTL)
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
		return s.opts.JWTSecret, nil
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

// UpdateProfile changes name and email. A new email is unverified until its link is opened.
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
	if !user.EmailVerifiedAt.Valid {
		s.sendEmailVerification(ctx, user)
	}
	return &user, nil
}

// RequestPasswordReset emails a reset link when the address belongs to an account. It
// reports nothing about whether it does, so the endpoint cannot be used to find accounts.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return
	}

	token, err := s.issueEmailToken(ctx, user.ID, purposePasswordReset, s.opts.PasswordResetTTL)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to create password reset token", "error", err, "user_id", user.ID)
		return
	}
	if err := s.mail.Send(ctx, s.passwordResetEmail(user, token)); err != nil {
		s.logger.ErrorContext(ctx, "Failed to send password reset email", "error", err, "user_id", user.ID)
	}
}

// ResetPassword sets a new password from an emailed link and revokes every refresh token.
// Opening the link also proves the mailbox is theirs, so the email becomes verified.
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.tx.WithTx(ctx, func(q *db.Queries) error {
		userID, err := consumeEmailToken(ctx, q, token, purposePasswordReset)
		if err != nil {
			return err
		}
		if err := q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: userID, Password: string(hashedPassword)}); err != nil {
			return errs.WrapDatabaseError(err)
		}
		if err := q.MarkUserEmailVerified(ctx, userID); err != nil {
			return errs.WrapDatabaseError(err)
		}
		return errs.WrapDatabaseError(q.RevokeAllUserRefreshTokens(ctx, userID))
	})
}

// VerifyEmail confirms an email address from an emailed link.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	return s.tx.WithTx(ctx, func(q *db.Queries) error {
		userID, err := consumeEmailToken(ctx, q, token, purposeEmailVerification)
		if err != nil {
			return err
		}
		return errs.WrapDatabaseError(q.MarkUserEmailVerified(ctx, userID))
	})
}

// ResendEmailVerification sends a fresh link to the current user.
func (s *AuthService) ResendEmailVerification(ctx context.Context, userID int32) error {
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if user.EmailVerifiedAt.Valid {
		return ErrEmailAlreadyVerified
	}

	token, err := s.issueEmailToken(ctx, user.ID, purposeEmailVerification, s.opts.EmailVerificationTTL)
	if err != nil {
		return err
	}
	if err := s.mail.Send(ctx, s.emailVerificationEmail(user, token)); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}
	return nil
}

// sendEmailVerification is best effort: an account must not fail to be created or updated
// because the mail server is down. The user can ask for another link later.
func (s *AuthService) sendEmailVerification(ctx context.Context, user db.User) {
	token, err := s.issueEmailToken(ctx, user.ID, purposeEmailVerification, s.opts.EmailVerificationTTL)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to create email verification token", "error", err, "user_id", user.ID)
		return
	}
	if err := s.mail.Send(ctx, s.emailVerificationEmail(user, token)); err != nil {
		s.logger.ErrorContext(ctx, "Failed to send verification email", "error", err, "user_id", user.ID)
	}
}

// issueEmailToken replaces any earlier token of the same purpose, so only the newest link works.
func (s *AuthService) issueEmailToken(ctx context.Context, userID int32, purpose string, ttl time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(raw)

	err := s.tx.WithTx(ctx, func(q *db.Queries) error {
		if err := q.DeleteUserAuthTokens(ctx, db.DeleteUserAuthTokensParams{UserID: userID, Purpose: purpose}); err != nil {
			return errs.WrapDatabaseError(err)
		}
		return errs.WrapDatabaseError(q.CreateAuthToken(ctx, db.CreateAuthTokenParams{
			UserID:     userID,
			Purpose:    purpose,
			TokenHash:  hashEmailToken(token),
			TtlSeconds: int32(ttl.Seconds()),
		}))
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func consumeEmailToken(ctx context.Context, q *db.Queries, token, purpose string) (int32, error) {
	userID, err := q.ConsumeAuthToken(ctx, db.ConsumeAuthTokenParams{TokenHash: hashEmailToken(token), Purpose: purpose})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInvalidEmailToken
	}
	if err != nil {
		return 0, errs.WrapDatabaseError(err)
	}
	return userID, nil
}

func hashEmailToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
