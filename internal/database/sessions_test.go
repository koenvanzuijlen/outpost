package database

import (
	"errors"
	"testing"
	"time"
)

var sessionT0 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// newSessionStore returns a store with the admin "koen" and the session "token-a" created at sessionT0.
func newSessionStore(t *testing.T) *Store {
	t.Helper()
	store := newTestStore(t)
	ctx := t.Context()
	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	if _, err := store.CreateSession(ctx, 1, "token-a", sessionT0); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	return store
}

// TestSessionUser checks expiry, renewal and lookup of sessions.
func TestSessionUser(t *testing.T) {
	t0 := sessionT0
	tests := []struct {
		name        string
		token       string
		now         time.Time
		wantErr     error
		wantRenewed bool
		wantExpires time.Time
	}{
		{"fresh", "token-a", t0.Add(time.Hour), nil, false, time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)},
		{"renewal due", "token-a", t0.Add(25 * time.Hour), nil, true, time.Date(2026, 2, 1, 13, 0, 0, 0, time.UTC)},
		{"expired", "token-a", t0.Add(30 * 24 * time.Hour), ErrSessionNotFound, false, time.Time{}},
		{"unknown", "token-b", t0.Add(time.Hour), ErrSessionNotFound, false, time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newSessionStore(t)
			ctx := t.Context()

			session, err := store.SessionUser(ctx, tt.token, tt.now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("SessionUser() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if session.User.Username != "koen" {
				t.Errorf("SessionUser() username = %q, want %q", session.User.Username, "koen")
			}
			if session.Renewed != tt.wantRenewed {
				t.Errorf("SessionUser() renewed = %v, want %v", session.Renewed, tt.wantRenewed)
			}
			if !session.ExpiresAt.Equal(tt.wantExpires) {
				t.Errorf("SessionUser() expires at = %v, want %v", session.ExpiresAt, tt.wantExpires)
			}

			if !tt.wantRenewed {
				return
			}
			again, err := store.SessionUser(ctx, tt.token, t0.Add(26*time.Hour))
			if err != nil {
				t.Fatalf("second SessionUser() error = %v", err)
			}
			if !again.ExpiresAt.Equal(tt.wantExpires) {
				t.Errorf("second SessionUser() expires at = %v, want %v", again.ExpiresAt, tt.wantExpires)
			}
			if again.Renewed {
				t.Errorf("second SessionUser() renewed = %v, want %v", again.Renewed, false)
			}
		})
	}
}

// TestSessionRenewalDue checks the renewal threshold.
func TestSessionRenewalDue(t *testing.T) {
	now := sessionT0
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"full lifetime", now.Add(30 * 24 * time.Hour), false},
		{"29 days left", now.Add(29 * 24 * time.Hour), false},
		{"29 days left minus 1s", now.Add(29*24*time.Hour - time.Second), true},
		{"1 hour left", now.Add(time.Hour), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sessionRenewalDue(tt.expiresAt, now)
			if got != tt.want {
				t.Errorf("sessionRenewalDue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCreateSessionDeletesExpired checks that CreateSession removes expired sessions.
func TestCreateSessionDeletesExpired(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}
	if _, err := store.CreateSession(ctx, 1, "old", sessionT0); err != nil {
		t.Fatalf("CreateSession(old) error = %v", err)
	}
	if _, err := store.CreateSession(ctx, 1, "new", sessionT0.Add(31*24*time.Hour)); err != nil {
		t.Fatalf("CreateSession(new) error = %v", err)
	}

	var count int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 1 {
		t.Errorf("session count = %d, want %d", count, 1)
	}
}

// TestDeleteSession checks that DeleteSession removes a session and that deleting again succeeds.
func TestDeleteSession(t *testing.T) {
	store := newSessionStore(t)
	ctx := t.Context()

	if err := store.DeleteSession(ctx, "token-a"); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	_, err := store.SessionUser(ctx, "token-a", sessionT0.Add(time.Hour))
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("SessionUser() error = %v, want %v", err, ErrSessionNotFound)
	}
	if err := store.DeleteSession(ctx, "token-a"); err != nil {
		t.Errorf("second DeleteSession() error = %v, want nil", err)
	}
}

// TestSessionNonUTC checks that session times in other zones compare by instant, not by text.
func TestSessionNonUTC(t *testing.T) {
	zone := time.FixedZone("x", 14*3600)
	store := newTestStore(t)
	ctx := t.Context()
	if _, err := store.CompleteSetup(ctx, "koen", "hash"); err != nil {
		t.Fatalf("CompleteSetup() error = %v", err)
	}

	expiresAt, err := store.CreateSession(ctx, 1, "old", sessionT0.In(zone))
	if err != nil {
		t.Fatalf("CreateSession(old) error = %v", err)
	}
	if want := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC); !expiresAt.Equal(want) {
		t.Errorf("CreateSession() expires at = %v, want %v", expiresAt, want)
	}

	// One hour after the expiry instant, written in a zone that sorts before the stored text.
	later := time.Date(2026, 1, 31, 13, 0, 0, 0, time.UTC).In(time.FixedZone("y", -12*3600))
	_, err = store.SessionUser(ctx, "old", later)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("SessionUser() error = %v, want %v", err, ErrSessionNotFound)
	}

	if _, err := store.CreateSession(ctx, 1, "new", later); err != nil {
		t.Fatalf("CreateSession(new) error = %v", err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 1 {
		t.Errorf("session count = %d, want %d", count, 1)
	}
}
