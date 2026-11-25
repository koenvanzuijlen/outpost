package server

import (
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// TestUserLocation checks that a user without a time zone, or with a name that does not load, gets the server time
// zone.
func TestUserLocation(t *testing.T) {
	tests := []struct {
		name     string
		timeZone *string
		want     string
	}{
		{name: "none", timeZone: nil, want: time.Local.String()},
		// The zone must differ from TZ, or the row cannot tell a loaded zone from the fallback.
		{name: "valid", timeZone: new("Pacific/Kiritimati"), want: "Pacific/Kiritimati"},
		{name: "unknown", timeZone: new("Mars/Olympus_Mons"), want: time.Local.String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userLocation(database.User{TimeZone: tt.timeZone}).String()
			if got != tt.want {
				t.Errorf("userLocation() = %q, want %q", got, tt.want)
			}
		})
	}
}
