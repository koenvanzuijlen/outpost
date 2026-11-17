package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// setupUserID is the user that the first migration seeds. The setup turns this user into the first admin,
// so the library and the wishlist from before user accounts stay with that admin.
const setupUserID = 1

// ErrUserNotFound is returned when no user has the given username or ID.
var ErrUserNotFound = errors.New("user not found")

// ErrSetupDone is returned when the setup runs while an admin already exists.
var ErrSetupDone = errors.New("setup already done")

// UserRole is the role of a user, one of the UserRole constants. Values are stored as text and validated in Go.
type UserRole string

// The user roles. An admin manages the accounts of the other users.
const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

// User is a user account without its password hash.
type User struct {
	ID       int64
	Username string
	Role     UserRole
}

// IsAdmin reports whether u has the admin role.
func (u User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// userFrom converts a users row to a User and leaves out the password hash.
func userFrom(row dbgen.User) User {
	return User{
		ID:       row.ID,
		Username: row.Username,
		Role:     UserRole(row.Role),
	}
}

// SetupNeeded reports whether no admin exists yet. Until an admin exists, the setup page creates the first admin.
func (s *Store) SetupNeeded(ctx context.Context) (bool, error) {
	admincount, err := s.queries.CountUsersWithRole(ctx, string(UserRoleAdmin))
	if err != nil {
		return false, fmt.Errorf("count admins: %w", err)
	}
	return admincount == 0, nil
}

// CompleteSetup turns the seeded user into the first admin, with username and passwordHash.
// The library and the wishlist of the seeded user stay with the admin.
// Returns ErrSetupDone if an admin already exists.
func (s *Store) CompleteSetup(ctx context.Context, username, passwordHash string) (User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	admincount, err := q.CountUsersWithRole(ctx, string(UserRoleAdmin))
	if err != nil {
		return User{}, fmt.Errorf("count admins: %w", err)
	}
	if admincount > 0 {
		return User{}, ErrSetupDone
	}

	rows, err := q.UpdateUserForSetup(ctx, dbgen.UpdateUserForSetupParams{
		ID:           setupUserID,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         string(UserRoleAdmin),
	})
	if err != nil {
		return User{}, fmt.Errorf("update user for setup: %w", err)
	}
	if rows == 0 {
		return User{}, fmt.Errorf("seeded user %d not found", setupUserID)
	}

	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit tx: %w", err)
	}
	return User{ID: setupUserID, Username: username, Role: UserRoleAdmin}, nil
}

// UserByUsername returns the user with username and the password hash of that user.
// Returns ErrUserNotFound if no user has that username.
func (s *Store) UserByUsername(ctx context.Context, username string) (User, string, error) {
	row, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, "", ErrUserNotFound
		}
		return User{}, "", fmt.Errorf("get user: %w", err)
	}
	return userFrom(row), row.PasswordHash, nil
}
