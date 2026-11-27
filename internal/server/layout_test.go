package server

import (
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// TestLayoutTheme checks the data-theme attribute of the html element. A signed-in user gets the stored theme on every
// page. A page without a user gets the theme "system".
func TestLayoutTheme(t *testing.T) {
	tests := []struct {
		name      string
		signedOut bool
		stored    database.Theme
		path      string
		want      string
	}{
		{"default", false, "", "/settings/account", `<html lang="en" data-theme="system">`},
		{"dark", false, database.ThemeDark, "/settings/account", `<html lang="en" data-theme="dark">`},
		{"light", false, database.ThemeLight, "/", `<html lang="en" data-theme="light">`},
		{"signed out", true, "", "/sign-in", `<html lang="en" data-theme="system">`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *Server
			if tt.signedOut {
				server = newSignedOutTestServer(t)
			} else {
				server = newTestServer(t)
			}
			if tt.stored != "" {
				if err := server.store.SetUserTheme(t.Context(), testUserID, tt.stored); err != nil {
					t.Fatalf("SetUserTheme() error = %v", err)
				}
			}

			w := getPage(t, server, tt.path)
			assertBodyContains(t, w, tt.want)
		})
	}
}
