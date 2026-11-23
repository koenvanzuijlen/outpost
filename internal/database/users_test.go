package database

import (
	"errors"
	"slices"
	"testing"
	"time"
)

// TestSetupNeeded checks that setup is needed until CompleteSetup runs.
func TestSetupNeeded(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	got, err := store.SetupNeeded(ctx)
	if err != nil {
		t.Fatalf("SetupNeeded() error = %v", err)
	}
	if !got {
		t.Errorf("SetupNeeded() = %v, want %v", got, true)
	}

	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}

	got, err = store.SetupNeeded(ctx)
	if err != nil {
		t.Fatalf("SetupNeeded() error = %v", err)
	}
	if got {
		t.Errorf("SetupNeeded() = %v, want %v", got, false)
	}
}

// TestCompleteSetup checks that setup works once and that a second setup does not create a user.
func TestCompleteSetup(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	want := User{ID: 1, Username: "koen", Role: "admin"}

	user, err := store.CompleteSetup(ctx, "koen", "hash")
	if err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	if user != want {
		t.Errorf("CompleteSetup() = %v, want %v", user, want)
	}

	found, hash, err := store.UserByUsername(ctx, "koen")
	if err != nil {
		t.Fatalf("UserByUsername() error = %v", err)
	}
	if found != want {
		t.Errorf("UserByUsername() user = %v, want %v", found, want)
	}
	if hash != "hash" {
		t.Errorf("UserByUsername() hash = %q, want %q", hash, "hash")
	}

	_, err = store.CompleteSetup(ctx, "other", "hash2")
	if !errors.Is(err, ErrSetupDone) {
		t.Errorf("second CompleteSetup() error = %v, want %v", err, ErrSetupDone)
	}

	_, _, err = store.UserByUsername(ctx, "other")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UserByUsername(other) error = %v, want %v", err, ErrUserNotFound)
	}
}

// TestCompleteSetupKeepsLibrary checks that setup keeps the library of the seeded user.
func TestCompleteSetupKeepsLibrary(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), nil, 1, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddToLibrary() error = %v", err)
	}
	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}

	entries, err := store.ListLibrary(ctx, 1)
	if err != nil {
		t.Fatalf("ListLibrary() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListLibrary() length = %d, want %d", len(entries), 1)
	}
	if entries[0].Game.IgdbID != 100 {
		t.Errorf("ListLibrary() IGDB ID = %d, want %d", entries[0].Game.IgdbID, 100)
	}
}

// TestUserByUsernameNotFound checks that an unknown username returns ErrUserNotFound.
func TestUserByUsernameNotFound(t *testing.T) {
	store := newTestStore(t)

	_, _, err := store.UserByUsername(t.Context(), "nobody")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UserByUsername() error = %v, want %v", err, ErrUserNotFound)
	}
}

// newTwoUserStore returns a store with the admin "koen" (ID 1) and the user "anna" (ID 2).
func newTwoUserStore(t *testing.T) *Store {
	t.Helper()
	store := newTestStore(t)
	ctx := t.Context()

	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	_, err := store.db.ExecContext(ctx, "INSERT INTO users(id, username, password_hash) VALUES (2, 'anna', '!')")
	if err != nil {
		t.Fatalf("insert user anna: %v", err)
	}
	return store
}

// TestRenameUser checks which renames succeed. The own username counts as success.
func TestRenameUser(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		username string
		wantErr  error
	}{
		{name: "new name", userID: 1, username: "koen2", wantErr: nil},
		{name: "same name", userID: 1, username: "koen", wantErr: nil},
		{name: "taken", userID: 1, username: "anna", wantErr: ErrUsernameTaken},
		{name: "unknown user", userID: 99, username: "nobody", wantErr: ErrUserNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTwoUserStore(t)

			err := store.RenameUser(t.Context(), tt.userID, tt.username)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("RenameUser() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestRenameUserStoresUsername checks that the user has the new username after a rename.
func TestRenameUserStoresUsername(t *testing.T) {
	store := newTwoUserStore(t)
	ctx := t.Context()

	if err := store.RenameUser(ctx, 1, "koen2"); err != nil {
		t.Fatalf("RenameUser() error = %v", err)
	}

	user, _, err := store.UserByUsername(ctx, "koen2")
	if err != nil {
		t.Fatalf("UserByUsername() error = %v", err)
	}
	if user.ID != 1 {
		t.Errorf("UserByUsername() ID = %d, want %d", user.ID, 1)
	}
}

// TestChangePassword checks that a password change stores the new hash and deletes only the other sessions of
// the user.
func TestChangePassword(t *testing.T) {
	store := newTwoUserStore(t)
	ctx := t.Context()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	sessions := []struct {
		userID    int64
		tokenHash string
	}{
		{userID: 1, tokenHash: "keep"},
		{userID: 1, tokenHash: "other"},
		{userID: 2, tokenHash: "anna"},
	}
	for _, s := range sessions {
		if _, err := store.CreateSession(ctx, s.userID, s.tokenHash, t0); err != nil {
			t.Fatalf("CreateSession(%q) error = %v", s.tokenHash, err)
		}
	}

	if err := store.ChangePassword(ctx, 1, "new-hash", "keep"); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	hash, err := store.PasswordHash(ctx, 1)
	if err != nil {
		t.Fatalf("PasswordHash() error = %v", err)
	}
	if hash != "new-hash" {
		t.Errorf("PasswordHash() = %q, want %q", hash, "new-hash")
	}

	tests := []struct {
		tokenHash string
		wantErr   error
	}{
		{tokenHash: "keep", wantErr: nil},
		{tokenHash: "other", wantErr: ErrSessionNotFound},
		{tokenHash: "anna", wantErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.tokenHash, func(t *testing.T) {
			_, err := store.SessionUser(ctx, tt.tokenHash, t0.Add(time.Hour))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SessionUser(%q) error = %v, want %v", tt.tokenHash, err, tt.wantErr)
			}
		})
	}
}

// TestChangePasswordAllSessions checks that an empty keepTokenHash deletes all sessions of the user.
func TestChangePasswordAllSessions(t *testing.T) {
	store := newTwoUserStore(t)
	ctx := t.Context()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	if _, err := store.CreateSession(ctx, 1, "keep", t0); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.ChangePassword(ctx, 1, "new-hash", ""); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}

	_, err := store.SessionUser(ctx, "keep", t0.Add(time.Hour))
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("SessionUser() error = %v, want %v", err, ErrSessionNotFound)
	}
}

// TestChangePasswordUnknownUser checks that a password change of an unknown user returns ErrUserNotFound.
func TestChangePasswordUnknownUser(t *testing.T) {
	store := newTwoUserStore(t)

	err := store.ChangePassword(t.Context(), 99, "h", "")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("ChangePassword() error = %v, want %v", err, ErrUserNotFound)
	}
}

// TestPasswordHashUnknownUser checks that the password hash of an unknown user returns ErrUserNotFound.
func TestPasswordHashUnknownUser(t *testing.T) {
	store := newTwoUserStore(t)

	_, err := store.PasswordHash(t.Context(), 99)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("PasswordHash() error = %v, want %v", err, ErrUserNotFound)
	}
}

// TestCreateUser checks which users can be created. The rows run in order on one store.
func TestCreateUser(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}

	tests := []struct {
		name     string
		username string
		role     UserRole
		wantErr  error
	}{
		{name: "user", username: "anna", role: "user", wantErr: nil},
		{name: "admin", username: "bram", role: "admin", wantErr: nil},
		{name: "taken", username: "koen", role: "user", wantErr: ErrUsernameTaken},
		{name: "invalid role", username: "carl", role: "owner", wantErr: ErrInvalidUserRole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := store.CreateUser(ctx, tt.username, "hash2", tt.role)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateUser() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			found, hash, err := store.UserByUsername(ctx, tt.username)
			if err != nil {
				t.Fatalf("UserByUsername() error = %v", err)
			}
			if found != created {
				t.Errorf("UserByUsername() user = %v, want %v", found, created)
			}
			if found.Role != tt.role {
				t.Errorf("UserByUsername() role = %q, want %q", found.Role, tt.role)
			}
			if hash != "hash2" {
				t.Errorf("UserByUsername() hash = %q, want %q", hash, "hash2")
			}
		})
	}
}

// TestListUsers checks that ListUsers returns all users, sorted by username.
func TestListUsers(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	for _, username := range []string{"zoe", "anna"} {
		if _, err := store.CreateUser(ctx, username, "hash2", UserRoleUser); err != nil {
			t.Fatalf("CreateUser(%q) error = %v", username, err)
		}
	}

	users, err := store.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	got := make([]string, 0, len(users))
	for _, user := range users {
		got = append(got, user.Username)
	}
	want := []string{"anna", "koen", "zoe"}
	if !slices.Equal(got, want) {
		t.Errorf("ListUsers() usernames = %v, want %v", got, want)
	}
}

// TestSetUserRole checks which role changes succeed.
func TestSetUserRole(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		role    UserRole
		wantErr error
	}{
		{name: "make admin", userID: 2, role: "admin", wantErr: nil},
		{name: "same role", userID: 2, role: "user", wantErr: nil},
		{name: "invalid", userID: 2, role: "owner", wantErr: ErrInvalidUserRole},
		{name: "unknown", userID: 99, role: "user", wantErr: ErrUserNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTwoUserStore(t)
			ctx := t.Context()

			err := store.SetUserRole(ctx, tt.userID, tt.role)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SetUserRole() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			user, _, err := store.UserByUsername(ctx, "anna")
			if err != nil {
				t.Fatalf("UserByUsername() error = %v", err)
			}
			if user.Role != tt.role {
				t.Errorf("UserByUsername() role = %q, want %q", user.Role, tt.role)
			}
		})
	}
}

// TestDeleteUser checks that a delete removes all data of the user and keeps the data of the other users and the
// catalog.
func TestDeleteUser(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	anna, err := store.CreateUser(ctx, "anna", "hash2", UserRoleUser)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	err = store.AddToLibrary(ctx, newGameParams(100, "Test Game"), new(newPlatformParams(11, "Xbox")), anna.ID, now)
	if err != nil {
		t.Fatalf("AddToLibrary(anna) error = %v", err)
	}
	if err := store.AddToLibrary(ctx, newGameParams(100, "Test Game"), nil, 1, now); err != nil {
		t.Fatalf("AddToLibrary(koen) error = %v", err)
	}
	if _, err := store.StartPlaythrough(ctx, anna.ID, 100, nil, nil, now); err != nil {
		t.Fatalf("StartPlaythrough() error = %v", err)
	}
	if _, err := store.CreateSession(ctx, anna.ID, "anna-token", now); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if err := store.DeleteUser(ctx, anna.ID); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	tests := []struct {
		query string
		want  int
	}{
		{query: "SELECT COUNT(*) FROM users WHERE username = 'anna'", want: 0},
		{query: "SELECT COUNT(*) FROM user_games", want: 1},
		{query: "SELECT COUNT(*) FROM owned_copies", want: 0},
		{query: "SELECT COUNT(*) FROM playthroughs", want: 0},
		{query: "SELECT COUNT(*) FROM sessions", want: 0},
		{query: "SELECT COUNT(*) FROM games WHERE igdb_id = 100", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			var got int
			if err := store.db.QueryRowContext(ctx, tt.query).Scan(&got); err != nil {
				t.Fatalf("QueryRowContext() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("%s = %d, want %d", tt.query, got, tt.want)
			}
		})
	}

	err = store.DeleteUser(ctx, anna.ID)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("second DeleteUser() error = %v, want %v", err, ErrUserNotFound)
	}
}
