package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// SessionLifetime is how long a session stays valid after it was created or last renewed.
const SessionLifetime = 30 * 24 * time.Hour

// sessionRenewalInterval is the minimum time between two renewals of a session,
// so most requests do not write to the database.
const sessionRenewalInterval = 24 * time.Hour

// ErrSessionNotFound is returned when no session has the given token hash, or the session has expired.
var ErrSessionNotFound = errors.New("session not found")

// Session is a valid session with its user. Renewed reports whether SessionUser extended the session,
// so the caller sends the cookie again with the new ExpiresAt.
type Session struct {
	User      User
	ExpiresAt time.Time
	Renewed   bool
}

// CreateSession stores a new session of the user with tokenHash and returns the time it expires.
// It also deletes the expired sessions of all users.
func (s *Store) CreateSession(ctx context.Context, userID int64, tokenHash string, now time.Time) (time.Time, error) {
	now = now.UTC()
	expiresAt := now.Add(SessionLifetime)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := dbgen.New(tx)

	err = q.DeleteExpiredSessions(ctx, now)
	if err != nil {
		return time.Time{}, fmt.Errorf("delete expired sessions: %w", err)
	}

	err = q.InsertSession(ctx, dbgen.InsertSessionParams{
		TokenHash: tokenHash,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("insert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return time.Time{}, fmt.Errorf("commit tx: %w", err)
	}
	return expiresAt, nil
}

// SessionUser returns the session with tokenHash and its user. It renews the session to now plus SessionLifetime
// when the last renewal is more than a day ago.
// Returns ErrSessionNotFound if the session does not exist or has expired.
func (s *Store) SessionUser(ctx context.Context, tokenHash string, now time.Time) (Session, error) {
	now = now.UTC()

	row, err := s.queries.GetSessionUser(ctx, dbgen.GetSessionUserParams{
		TokenHash: tokenHash,
		Now:       now,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("get session user: %w", err)
	}

	session := Session{
		User:      userFrom(row.User),
		ExpiresAt: row.ExpiresAt,
		Renewed:   false,
	}
	if sessionRenewalDue(session.ExpiresAt, now) {
		session.ExpiresAt = now.Add(SessionLifetime)
		err = s.queries.UpdateSessionExpiry(ctx, dbgen.UpdateSessionExpiryParams{
			TokenHash: tokenHash,
			ExpiresAt: session.ExpiresAt,
		})
		if err != nil {
			return Session{}, fmt.Errorf("renew session: %w", err)
		}
		session.Renewed = true
	}
	return session, nil
}

// sessionRenewalDue reports whether more than sessionRenewalInterval has passed since the session
// that expires at expiresAt was created or renewed.
func sessionRenewalDue(expiresAt, now time.Time) bool {
	return expiresAt.Sub(now) < SessionLifetime-sessionRenewalInterval
}

// DeleteSession deletes the session with tokenHash. A session that does not exist counts as success.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	err := s.queries.DeleteSession(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
