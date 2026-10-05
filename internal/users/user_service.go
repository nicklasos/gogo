package users

import (
	"context"
	"fmt"
	"slices"

	"app/internal/db"
	"app/internal/errs"
	"app/internal/middleware"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists = errs.NewBadRequestError(errs.ErrKeyAuthUserExists, "User with this email already exists")
	ErrUserNotFound      = errs.NewNotFoundError(errs.ErrKeyUsersNotFound, "User not found")
	ErrForbiddenRole     = errs.NewForbiddenError(errs.ErrKeyUsersForbiddenRole, "You cannot manage users with this role")
	ErrCannotDeleteSelf  = errs.NewBadRequestError(errs.ErrKeyUsersCannotDeleteSelf, "Cannot delete yourself")
)

type UserService struct {
	queries *db.Queries
	tx      *db.TxRunner
}

type PaginatedUsers struct {
	Data  []db.User
	Total int64
}

func NewUserService(queries *db.Queries, tx *db.TxRunner) *UserService {
	return &UserService{queries: queries, tx: tx}
}

// CanManage is the single rule for who may manage whom: a super admin manages everyone,
// an admin manages only accounts whose every role is "user".
func CanManage(callerRoles, targetRoles []string) bool {
	if slices.Contains(callerRoles, middleware.RoleSuperAdmin) {
		return true
	}
	if !slices.Contains(callerRoles, middleware.RoleAdmin) || len(targetRoles) == 0 {
		return false
	}
	for _, role := range targetRoles {
		if role != middleware.RoleUser {
			return false
		}
	}
	return true
}

func (s *UserService) ListByRole(ctx context.Context, callerRoles []string, role string, page, pageSize int32) (*PaginatedUsers, error) {
	if !CanManage(callerRoles, []string{role}) {
		return nil, ErrForbiddenRole
	}

	total, err := s.queries.CountUsersByRole(ctx, role)
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	users, err := s.queries.ListUsersByRole(ctx, db.ListUsersByRoleParams{
		Role:       role,
		PageLimit:  pageSize,
		PageOffset: (page - 1) * pageSize,
	})
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	return &PaginatedUsers{Data: users, Total: total}, nil
}

func (s *UserService) Create(ctx context.Context, callerRoles []string, req CreateUserRequest) (*db.User, error) {
	if !CanManage(callerRoles, []string{req.Role}) {
		return nil, ErrForbiddenRole
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashedPassword),
		Roles:    []string{req.Role},
		// The admin creating the account vouches for the address
		EmailVerified: true,
	})
	if err != nil {
		if domainErr := errs.DomainErrorFromPostgresUniqueViolation(err); domainErr != nil {
			return nil, ErrUserAlreadyExists
		}
		return nil, errs.WrapDatabaseError(err)
	}
	return &user, nil
}

func (s *UserService) Update(ctx context.Context, callerRoles []string, id int32, req UpdateUserRequest) (*db.User, error) {
	if err := s.requireManageable(ctx, callerRoles, id); err != nil {
		return nil, err
	}

	user, err := s.queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:           id,
		Email:        req.Email,
		Name:         req.Name,
		KeepVerified: true,
	})
	if err != nil {
		if domainErr := errs.DomainErrorFromPostgresUniqueViolation(err); domainErr != nil {
			return nil, ErrUserAlreadyExists
		}
		return nil, errs.WrapDatabaseError(err)
	}
	return &user, nil
}

// SetPassword replaces the password and revokes the user's refresh tokens.
func (s *UserService) SetPassword(ctx context.Context, callerRoles []string, id int32, password string) error {
	if err := s.requireManageable(ctx, callerRoles, id); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.tx.WithTx(ctx, func(q *db.Queries) error {
		if err := q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
			ID:       id,
			Password: string(hashedPassword),
		}); err != nil {
			return errs.WrapDatabaseError(err)
		}
		return errs.WrapDatabaseError(q.RevokeAllUserRefreshTokens(ctx, id))
	})
}

// Delete removes a user. Deleting yourself is refused, which also guarantees
// that the last super admin can never be deleted.
func (s *UserService) Delete(ctx context.Context, callerID int32, callerRoles []string, id int32) error {
	if id == callerID {
		return ErrCannotDeleteSelf
	}
	if err := s.requireManageable(ctx, callerRoles, id); err != nil {
		return err
	}
	if err := s.queries.DeleteUser(ctx, id); err != nil {
		return errs.WrapDatabaseError(err)
	}
	return nil
}

func (s *UserService) requireManageable(ctx context.Context, callerRoles []string, id int32) error {
	user, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}
	if !CanManage(callerRoles, user.Roles) {
		return ErrForbiddenRole
	}
	return nil
}
