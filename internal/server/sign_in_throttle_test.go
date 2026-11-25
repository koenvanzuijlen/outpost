package server

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// throttleT0 is the start time of the throttle tests.
var throttleT0 = time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

// TestSignInLockDuration tests the lock period for each number of attempts.
func TestSignInLockDuration(t *testing.T) {
	tests := []struct {
		count int
		want  time.Duration
	}{
		{5, time.Minute},
		{6, 2 * time.Minute},
		{7, 4 * time.Minute},
		{8, 8 * time.Minute},
		{9, 15 * time.Minute},
		{100, 15 * time.Minute},
	}
	for _, tt := range tests {
		got := signInLockDuration(tt.count)
		if got != tt.want {
			t.Errorf("signInLockDuration(%d) = %v, want %v", tt.count, got, tt.want)
		}
	}
}

// throttleStep is one call on a signInThrottle: reset(username) when reset is true, and otherwise
// attempt(username, throttleT0+at) with the expected wait.
type throttleStep struct {
	username string
	at       time.Duration
	reset    bool
	want     time.Duration
}

// TestSignInThrottle tests the count, the lock, the reset and forgetting of sign-in attempts.
func TestSignInThrottle(t *testing.T) {
	var fiveFree []throttleStep
	for i := range 5 {
		fiveFree = append(fiveFree, throttleStep{username: "admin", at: time.Duration(i) * time.Second})
	}

	tests := []struct {
		name  string
		steps []throttleStep
	}{
		{"lock after five attempts", []throttleStep{
			{username: "admin", at: 5 * time.Second, want: 59 * time.Second},
			{username: "admin", at: 64 * time.Second, want: 0},
			{username: "admin", at: 65 * time.Second, want: 119 * time.Second},
		}},
		{"locked attempt is not counted", []throttleStep{
			{username: "admin", at: 5 * time.Second, want: 59 * time.Second},
			{username: "admin", at: 30 * time.Second, want: 34 * time.Second},
			{username: "admin", at: 64 * time.Second, want: 0},
		}},
		{"lock grows to the maximum", []throttleStep{
			{username: "admin", at: 64 * time.Second, want: 0},
			{username: "admin", at: 184 * time.Second, want: 0},
			{username: "admin", at: 424 * time.Second, want: 0},
			{username: "admin", at: 904 * time.Second, want: 0},
			{username: "admin", at: 905 * time.Second, want: 899 * time.Second},
		}},
		{"reset clears the count", []throttleStep{
			{username: "admin", reset: true},
			{username: "admin", at: 5 * time.Second, want: 0},
		}},
		{"count is forgotten after an hour", []throttleStep{
			{username: "koen", at: time.Hour, want: 0},
			{username: "admin", at: time.Hour + 4*time.Second, want: 0},
			{username: "admin", at: time.Hour + 5*time.Second, want: 0},
		}},
		{"other username", []throttleStep{
			{username: "koen", at: 5 * time.Second, want: 0},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			throttle := newSignInThrottle()
			for _, step := range slices.Concat(fiveFree, tt.steps) {
				if step.reset {
					throttle.reset(step.username)
					continue
				}
				got := throttle.attempt(step.username, throttleT0.Add(step.at))
				if got != step.want {
					t.Errorf("attempt(%q, t0+%v) = %v, want %v", step.username, step.at, got, step.want)
				}
			}
		})
	}
}

// TestSignInThrottleConcurrent tests that parallel attempts cannot pass the number of free attempts.
func TestSignInThrottleConcurrent(t *testing.T) {
	throttle := newSignInThrottle()
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if throttle.attempt("admin", throttleT0) == 0 {
				allowed.Add(1)
			}
		})
	}
	wg.Wait()

	if got := allowed.Load(); got != 5 {
		t.Errorf("allowed attempts = %d, want %d", got, 5)
	}
}

// TestSignInThrottleSweep tests that an attempt removes the counts that are older than signInForgetAfter.
func TestSignInThrottleSweep(t *testing.T) {
	throttle := newSignInThrottle()
	throttle.attempt("admin", throttleT0)
	throttle.attempt("koen", throttleT0.Add(time.Hour))

	if got := len(throttle.attempts); got != 1 {
		t.Errorf("len(attempts) = %d, want %d", got, 1)
	}
	if _, ok := throttle.attempts["admin"]; ok {
		t.Errorf("attempts[%q] present = %v, want %v", "admin", ok, false)
	}
}

// TestSignInLockedMessage tests that the lock message rounds the wait up to whole minutes.
func TestSignInLockedMessage(t *testing.T) {
	tests := []struct {
		wait time.Duration
		want string
	}{
		{59 * time.Second, "Too many failed attempts. Try again in 1 minute."},
		{time.Minute, "Too many failed attempts. Try again in 1 minute."},
		{61 * time.Second, "Too many failed attempts. Try again in 2 minutes."},
		{15 * time.Minute, "Too many failed attempts. Try again in 15 minutes."},
	}
	for _, tt := range tests {
		got := signInLockedMessage(tt.wait)
		if got != tt.want {
			t.Errorf("signInLockedMessage(%v) = %q, want %q", tt.wait, got, tt.want)
		}
	}
}
