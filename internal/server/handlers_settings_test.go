package server

import (
	"net/http"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/auth"
)

// TestHandleSettingsSaved checks which success message each settings page shows for the saved query value. A message
// shows only on the page of its form.
func TestHandleSettingsSaved(t *testing.T) {
	tests := []struct {
		path       string
		query      string
		wantBody   string
		unwantBody string
	}{
		{path: "/settings/account", query: "?saved=username", wantBody: "Username saved."},
		{path: "/settings/account", query: "?saved=password", wantBody: "Password changed."},
		{path: "/settings/account", query: "?saved=time-zone", unwantBody: `role="status"`},
		{path: "/settings/account", query: "?saved=bogus", unwantBody: `role="status"`},
		{path: "/settings/preferences", query: "?saved=time-zone", wantBody: "Time zone saved."},
		{path: "/settings/preferences", query: "?saved=username", unwantBody: `role="status"`},
	}
	for _, tt := range tests {
		t.Run(tt.path+tt.query, func(t *testing.T) {
			server := newTestServer(t)

			w := getPage(t, server, tt.path+tt.query)
			if tt.wantBody != "" {
				assertBodyContains(t, w, tt.wantBody)
			}
			if tt.unwantBody != "" {
				assertBodyNotContains(t, w, tt.unwantBody)
			}
		})
	}
}

// TestHandleChangeUsername checks the validation and the redirect of a username change. The row "taken" creates the
// user anna first.
func TestHandleChangeUsername(t *testing.T) {
	tests := []struct {
		name         string
		otherUser    string
		body         string
		wantStatus   int
		wantLocation string
	}{
		{"valid", "", "username=Admin2", http.StatusSeeOther, "/settings/account?saved=username"},
		{"same", "", "username=admin", http.StatusSeeOther, "/settings/account?saved=username"},
		{"invalid", "", "username=a", http.StatusBadRequest, ""},
		{"taken", "anna", "username=anna", http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			if tt.otherUser != "" {
				addSignedInUser(t, server, tt.otherUser)
			}

			w := postForm(t, server, "/settings/account/username", tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
		})
	}
}

// TestHandleSetTimeZone checks the validation, the redirect and the stored value of a time zone change.
func TestHandleSetTimeZone(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantLocation string
		wantStored   *string
	}{
		{"zone", "time_zone=Europe%2FAmsterdam", http.StatusSeeOther, "/settings/preferences?saved=time-zone", new("Europe/Amsterdam")},
		{"spaces", "time_zone=+UTC+", http.StatusSeeOther, "/settings/preferences?saved=time-zone", new("UTC")},
		{"empty", "time_zone=", http.StatusSeeOther, "/settings/preferences?saved=time-zone", nil},
		{"unknown", "time_zone=Mars%2FOlympus_Mons", http.StatusBadRequest, "", nil},
		{"local", "time_zone=Local", http.StatusBadRequest, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)

			w := postForm(t, server, "/settings/preferences/time-zone", tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)

			user, _, err := server.store.UserByUsername(t.Context(), "admin")
			if err != nil {
				t.Fatalf("UserByUsername() error = %v", err)
			}
			got := user.TimeZone
			if (got == nil) != (tt.wantStored == nil) || (got != nil && *got != *tt.wantStored) {
				t.Errorf("UserByUsername() time zone = %v, want %v", derefOrNil(got), derefOrNil(tt.wantStored))
			}
		})
	}
}

// TestHandleSettingsPreferencesTimeZone checks that the time zone field of the preferences page shows the stored time
// zone of the user.
func TestHandleSettingsPreferencesTimeZone(t *testing.T) {
	tests := []struct {
		name       string
		stored     *string
		wantFilled bool
	}{
		{"none", nil, false},
		{"stored", new("Europe/Amsterdam"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			if tt.stored != nil {
				err := server.store.SetUserTimeZone(t.Context(), testUserID, tt.stored)
				if err != nil {
					t.Fatalf("SetUserTimeZone() error = %v", err)
				}
			}

			w := getPage(t, server, "/settings/preferences")
			if w.Code != http.StatusOK {
				t.Fatalf("GET /settings/preferences status = %d, want %d", w.Code, http.StatusOK)
			}
			if tt.wantFilled {
				assertBodyContains(t, w, `value="Europe/Amsterdam"`)
			} else {
				assertBodyNotContains(t, w, `value="Europe/Amsterdam"`)
			}
		})
	}
}

// TestHandleChangeUsernameStoresNormalized checks that a username change stores the normalized username.
func TestHandleChangeUsernameStoresNormalized(t *testing.T) {
	server := newTestServer(t)

	w := postForm(t, server, "/settings/account/username", "username=Admin2")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	user, _, err := server.store.UserByUsername(t.Context(), "admin2")
	if err != nil {
		t.Fatalf("UserByUsername() error = %v", err)
	}
	if user.ID != testUserID {
		t.Errorf("UserByUsername() ID = %d, want %d", user.ID, testUserID)
	}
}

// TestHandleChangePassword checks the validation, the current password check and the redirect of a password change.
func TestHandleChangePassword(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantLocation string
	}{
		{
			"valid",
			"current_password=correct+horse+battery&new_password=new+password+123&new_password_confirm=new+password+123",
			http.StatusSeeOther, "/settings/account?saved=password",
		},
		{
			"wrong current",
			"current_password=wrong+password+1&new_password=new+password+123&new_password_confirm=new+password+123",
			http.StatusBadRequest, "",
		},
		{
			"short new",
			"current_password=correct+horse+battery&new_password=short&new_password_confirm=short",
			http.StatusBadRequest, "",
		},
		{
			"different repeat",
			"current_password=correct+horse+battery&new_password=new+password+123&new_password_confirm=new+password+124",
			http.StatusBadRequest, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)

			w := postForm(t, server, "/settings/account/password", tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
		})
	}
}

// TestHandleChangePasswordKeepsSession checks that a password change stores the new password and keeps the current
// session.
func TestHandleChangePasswordKeepsSession(t *testing.T) {
	server := newTestServer(t)

	body := "current_password=correct+horse+battery&new_password=new+password+123&new_password_confirm=new+password+123"
	w := postForm(t, server, "/settings/account/password", body)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	hash, err := server.store.PasswordHash(t.Context(), testUserID)
	if err != nil {
		t.Fatalf("PasswordHash() error = %v", err)
	}
	ok, err := auth.VerifyPassword(hash, "new password 123")
	if err != nil || !ok {
		t.Errorf("VerifyPassword(new password) = %v, %v, want true, nil", ok, err)
	}

	getPage(t, server, "/library")
}
