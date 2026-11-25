package server

import (
	"fmt"
	"sync"
	"time"
)

// The rules of the sign-in lock. The first signInFreeAttempts attempts of a username are free. After that, each
// attempt locks the username for a period that starts at signInFirstLock and doubles up to signInMaxLock. A count is
// forgotten signInForgetAfter after its last attempt.
const (
	signInFreeAttempts = 5
	signInFirstLock    = time.Minute
	signInMaxLock      = 15 * time.Minute
	signInForgetAfter  = time.Hour
)

// signInAttempts is the number of sign-in attempts of one username since the last success, and the time of the last
// attempt.
type signInAttempts struct {
	count int
	last  time.Time
}

// signInLockDuration returns how long a username is locked after count attempts: signInFirstLock after
// signInFreeAttempts attempts, doubled for each further attempt, and at most signInMaxLock.
func signInLockDuration(count int) time.Duration {
	lock := signInFirstLock
	for i := signInFreeAttempts; i < count && lock < signInMaxLock; i++ {
		lock *= 2
	}
	return min(lock, signInMaxLock)
}

// lockedUntil returns the end of the sign-in lock after a's attempts, or the zero time if the free attempts are not
// used up.
func (a signInAttempts) lockedUntil() time.Time {
	if a.count < signInFreeAttempts {
		return time.Time{}
	}
	return a.last.Add(signInLockDuration(a.count))
}

// signInThrottle counts the sign-in attempts per username and locks a username after too many attempts. It is safe
// for use by multiple goroutines. The counts are in memory, so a restart clears them.
type signInThrottle struct {
	mu        sync.Mutex
	attempts  map[string]signInAttempts
	lastSweep time.Time
}

// newSignInThrottle returns a signInThrottle without attempts.
func newSignInThrottle() *signInThrottle {
	return &signInThrottle{
		attempts: make(map[string]signInAttempts),
	}
}

// attempt reports how long username must wait before its next sign-in attempt at now. A result of 0 allows the
// attempt and counts it. A locked attempt is not counted. Call reset after a successful sign-in.
func (t *signInThrottle) attempt(username string, now time.Time) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sweep(now)

	a := t.attempts[username]
	if now.Sub(a.last) >= signInForgetAfter {
		a = signInAttempts{}
	}

	until := a.lockedUntil()
	if now.Before(until) {
		return until.Sub(now)
	}

	a.count++
	a.last = now
	t.attempts[username] = a
	return 0
}

// reset removes the attempts of username, after a successful sign-in.
func (t *signInThrottle) reset(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.attempts, username)
}

// sweep removes the attempts that are older than signInForgetAfter, at most once per signInForgetAfter. The caller
// holds t.mu.
func (t *signInThrottle) sweep(now time.Time) {
	if now.Sub(t.lastSweep) < signInForgetAfter {
		return
	}
	for username, a := range t.attempts {
		if now.Sub(a.last) >= signInForgetAfter {
			delete(t.attempts, username)
		}
	}
	t.lastSweep = now
}

// signInLockedMessage returns the message of the sign-in form for a username that is locked for wait. It rounds the
// wait up to whole minutes.
func signInLockedMessage(wait time.Duration) string {
	mins := int((wait + time.Minute - 1) / time.Minute)
	if mins == 1 {
		return "Too many failed attempts. Try again in 1 minute."
	}
	return fmt.Sprintf("Too many failed attempts. Try again in %d minutes.", mins)
}
