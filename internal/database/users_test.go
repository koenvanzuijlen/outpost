package database

import (
	"errors"
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
