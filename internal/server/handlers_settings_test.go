package server

import (
	"net/http"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/auth"
)

// TestHandleSettingsSaved checks which success message the settings page shows for the saved query value.
func TestHandleSettingsSaved(t *testing.T) {
	tests := []struct {
		query      string
		wantBody   string
		unwantBody string
	}{
		{query: "?saved=username", wantBody: "Username saved."},
		{query: "?saved=password", wantBody: "Password changed."},
		{query: "?saved=time-zone", wantBody: "Time zone saved."},
		{query: "?saved=bogus", unwantBody: `role="status"`},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			server := newTestServer(t)

			w := getPage(t, server, "/settings"+tt.query)
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
		{"valid", "", "username=Admin2", http.StatusSeeOther, "/settings?saved=username"},
		{"same", "", "username=admin", http.StatusSeeOther, "/settings?saved=username"},
		{"invalid", "", "username=a", http.StatusBadRequest, ""},
		{"taken", "anna", "username=anna", http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)
			if tt.otherUser != "" {
				addSignedInUser(t, server, tt.otherUser)
			}

			w := postForm(t, server, "/settings/username", tt.body)
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
		{"zone", "time_zone=Europe%2FAmsterdam", http.StatusSeeOther, "/settings?saved=time-zone", new("Europe/Amsterdam")},
		{"spaces", "time_zone=+UTC+", http.StatusSeeOther, "/settings?saved=time-zone", new("UTC")},
		{"empty", "time_zone=", http.StatusSeeOther, "/settings?saved=time-zone", nil},
		{"unknown", "time_zone=Mars%2FOlympus_Mons", http.StatusBadRequest, "", nil},
		{"local", "time_zone=Local", http.StatusBadRequest, "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)

			w := postForm(t, server, "/settings/time-zone", tt.body)
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

// TestHandleChangeUsernameStoresNormalized checks that a username change stores the normalized username.
func TestHandleChangeUsernameStoresNormalized(t *testing.T) {
	server := newTestServer(t)

	w := postForm(t, server, "/settings/username", "username=Admin2")
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
			http.StatusSeeOther, "/settings?saved=password",
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

			w := postForm(t, server, "/settings/password", tt.body)
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
	w := postForm(t, server, "/settings/password", body)
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
