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

// ErrUsernameTaken is returned when another user already has the username.
var ErrUsernameTaken = errors.New("username taken")

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

// RenameUser changes the username of the user.
// Returns ErrUsernameTaken if another user has username, or ErrUserNotFound if the user does not exist.
func (s *Store) RenameUser(ctx context.Context, userID int64, username string) error {
	rows, err := s.queries.UpdateUsername(ctx, dbgen.UpdateUsernameParams{
		ID:       userID,
		Username: username,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUsernameTaken
		}
		return fmt.Errorf("update username: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// PasswordHash returns the password hash of the user. Returns ErrUserNotFound if the user does not exist.
func (s *Store) PasswordHash(ctx context.Context, userID int64) (string, error) {
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("get user: %w", err)
	}
	return user.PasswordHash, nil
}

// ChangePassword stores passwordHash as the password of the user and deletes all sessions of the user except the
// session with keepTokenHash. An empty keepTokenHash deletes all sessions of the user.
// Returns ErrUserNotFound if the user does not exist.
func (s *Store) ChangePassword(ctx context.Context, userID int64, passwordHash, keepTokenHash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	rows, err := q.UpdatePasswordHash(ctx, dbgen.UpdatePasswordHashParams{
		ID:           userID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	err = q.DeleteOtherSessions(ctx, dbgen.DeleteOtherSessionsParams{
		UserID:        userID,
		KeepTokenHash: keepTokenHash,
	})
	if err != nil {
		return fmt.Errorf("delete other sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
