package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

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

// ErrInvalidUserRole is returned when a role is not one of UserRoles.
var ErrInvalidUserRole = errors.New("invalid user role")

// UserRole is the role of a user, one of the UserRole constants. Values are stored as text and validated in Go.
type UserRole string

// The user roles. An admin manages the accounts of the other users.
const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

// UserRoles lists the valid roles, in the order that forms show them.
var UserRoles = []UserRole{
	UserRoleUser,
	UserRoleAdmin,
}

// Valid reports whether r is one of UserRoles.
func (r UserRole) Valid() bool {
	return slices.Contains(UserRoles, r)
}

// User is a user account without its password hash.
// TimeZone is an IANA time zone name, or nil for the time zone of the server.
type User struct {
	ID       int64
	Username string
	Role     UserRole
	TimeZone *string
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
		TimeZone: row.TimeZone,
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

// ListUsers returns all users, sorted by username.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.queries.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	users := make([]User, 0, len(rows))
	for _, row := range rows {
		users = append(users, userFrom(row))
	}
	return users, nil
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

// CreateUser creates a user with username, passwordHash and role.
// Returns ErrInvalidUserRole if role is not valid, or ErrUsernameTaken if another user has username.
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string, role UserRole) (User, error) {
	if !role.Valid() {
		return User{}, ErrInvalidUserRole
	}
	row, err := s.queries.InsertUser(ctx, dbgen.InsertUserParams{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         string(role),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrUsernameTaken
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	return userFrom(row), nil
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

// SetUserRole changes the role of the user. The current role counts as success.
// Returns ErrInvalidUserRole if role is not valid, or ErrUserNotFound if the user does not exist.
func (s *Store) SetUserRole(ctx context.Context, userID int64, role UserRole) error {
	if !role.Valid() {
		return ErrInvalidUserRole
	}

	rows, err := s.queries.UpdateUserRole(ctx, dbgen.UpdateUserRoleParams{
		ID:   userID,
		Role: string(role),
	})
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// SetUserTimeZone stores timeZone as the time zone of the user. A nil timeZone selects the time zone of the server.
// The caller validates the name. Returns ErrUserNotFound if the user does not exist.
func (s *Store) SetUserTimeZone(ctx context.Context, userID int64, timeZone *string) error {
	rows, err := s.queries.UpdateUserTimeZone(ctx, dbgen.UpdateUserTimeZoneParams{
		ID:       userID,
		TimeZone: timeZone,
	})
	if err != nil {
		return fmt.Errorf("update user time zone: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// DeleteUser deletes the user with the library, the wishlist, the copies, the DLC ownership, the playthroughs and the
// sessions of the user. The catalog games and platforms stay. Returns ErrUserNotFound if the user does not exist.
func (s *Store) DeleteUser(ctx context.Context, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	err = q.DeleteUserGamesForUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete user games: %w", err)
	}

	rows, err := q.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
