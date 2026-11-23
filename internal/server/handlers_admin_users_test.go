package server

import (
	"net/http"
	"testing"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// TestHandleCreateUser checks which users the admin can create, and that a created user has the role of the form.
func TestHandleCreateUser(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantLocation string
		wantUsername string
		wantRole     database.UserRole
	}{
		{
			name:       "user",
			body:       "username=Anna&password=correct+horse+battery&role=user",
			wantStatus: http.StatusSeeOther, wantLocation: "/admin/users",
			wantUsername: "anna", wantRole: database.UserRoleUser,
		},
		{
			name:       "admin",
			body:       "username=bram&password=correct+horse+battery&role=admin",
			wantStatus: http.StatusSeeOther, wantLocation: "/admin/users",
			wantUsername: "bram", wantRole: database.UserRoleAdmin,
		},
		{
			name:       "taken",
			body:       "username=admin&password=correct+horse+battery&role=user",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
		{
			name:       "invalid username",
			body:       "username=a%21&password=correct+horse+battery&role=user",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
		{
			name:       "short password",
			body:       "username=anna&password=short&role=user",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
		{
			name:       "invalid role",
			body:       "username=anna&password=correct+horse+battery&role=owner",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
		{
			name:       "no role",
			body:       "username=anna&password=correct+horse+battery",
			wantStatus: http.StatusBadRequest, wantLocation: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t)

			w := postForm(t, server, "/admin/users", tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("POST /admin/users status = %d, want %d", w.Code, tt.wantStatus)
			}
			assertLocation(t, w, tt.wantLocation)
			if tt.wantUsername == "" {
				return
			}

			user, _, err := server.store.UserByUsername(t.Context(), tt.wantUsername)
			if err != nil {
				t.Fatalf("UserByUsername(%q) error = %v", tt.wantUsername, err)
			}
			if user.Role != tt.wantRole {
				t.Errorf("UserByUsername(%q) role = %q, want %q", tt.wantUsername, user.Role, tt.wantRole)
			}
		})
	}
}
