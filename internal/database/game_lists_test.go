package database

import (
	"errors"
	"testing"
)

// TestCreateGameList checks that a new list stores its name, its description and its owner.
func TestCreateGameList(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	list, err := store.CreateGameList(ctx, 1, GameListDetails{Name: "RPGs", Description: new("Long games")})
	if err != nil {
		t.Fatalf("CreateGameList() error = %v", err)
	}
	if list.ID <= 0 {
		t.Errorf("CreateGameList() ID = %d, want > 0", list.ID)
	}
	if list.UserID != 1 {
		t.Errorf("CreateGameList() UserID = %d, want %d", list.UserID, 1)
	}
	if list.Name != "RPGs" {
		t.Errorf("CreateGameList() Name = %q, want %q", list.Name, "RPGs")
	}
	if got := derefOrNil(list.Description); got != "Long games" {
		t.Errorf("CreateGameList() Description = %v, want %q", got, "Long games")
	}

	got, err := store.GetGameList(ctx, 1, list.ID)
	if err != nil {
		t.Fatalf("GetGameList() error = %v", err)
	}
	if got.Name != "RPGs" {
		t.Errorf("GetGameList() Name = %q, want %q", got.Name, "RPGs")
	}
	if desc := derefOrNil(got.Description); desc != "Long games" {
		t.Errorf("GetGameList() Description = %v, want %q", desc, "Long games")
	}
}

// TestCreateGameListNameTaken checks that the names of the lists of one user are unique without case, and that other
// users can use the same name.
func TestCreateGameListNameTaken(t *testing.T) {
	tests := []struct {
		name    string
		userID  int64
		newName string
		wantErr error
	}{
		{name: "same name", userID: 1, newName: "RPGs", wantErr: ErrGameListNameTaken},
		{name: "other case", userID: 1, newName: "rpgs", wantErr: ErrGameListNameTaken},
		{name: "other user", userID: 2, newName: "RPGs", wantErr: nil},
		{name: "other name", userID: 1, newName: "Co-op", wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTwoUserStore(t)
			createGameList(t, store, 1, "RPGs")

			_, err := store.CreateGameList(t.Context(), tt.userID, GameListDetails{Name: tt.newName})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CreateGameList(%q) error = %v, want %v", tt.newName, err, tt.wantErr)
			}
		})
	}
}

// TestGetGameListNotFound checks that a list of another user and an unknown ID both give ErrGameListNotFound.
func TestGetGameListNotFound(t *testing.T) {
	store := newTwoUserStore(t)
	list := createGameList(t, store, 1, "RPGs")
	if list.ID != 1 {
		t.Fatalf("createGameList() ID = %d, want %d", list.ID, 1)
	}

	tests := []struct {
		name    string
		userID  int64
		listID  int64
		wantErr error
	}{
		{name: "other user", userID: 2, listID: 1, wantErr: ErrGameListNotFound},
		{name: "unknown ID", userID: 1, listID: 99, wantErr: ErrGameListNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.GetGameList(t.Context(), tt.userID, tt.listID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetGameList(%d, %d) error = %v, want %v", tt.userID, tt.listID, err, tt.wantErr)
			}
		})
	}
}

// TestUpdateGameList checks which updates succeed, and that a failed update keeps the stored values.
func TestUpdateGameList(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		listID   int64
		details  GameListDetails
		wantErr  error
		wantName string
		wantDesc any
	}{
		{
			name:     "rename",
			userID:   1,
			listID:   1,
			details:  GameListDetails{Name: "Role-playing", Description: new("Long games")},
			wantErr:  nil,
			wantName: "Role-playing",
			wantDesc: "Long games",
		},
		{
			name:     "change case",
			userID:   1,
			listID:   1,
			details:  GameListDetails{Name: "rpgs"},
			wantErr:  nil,
			wantName: "rpgs",
			wantDesc: nil,
		},
		{
			name:     "name taken",
			userID:   1,
			listID:   1,
			details:  GameListDetails{Name: "co-op"},
			wantErr:  ErrGameListNameTaken,
			wantName: "RPGs",
			wantDesc: nil,
		},
		{
			name:     "other user",
			userID:   2,
			listID:   1,
			details:  GameListDetails{Name: "Mine"},
			wantErr:  ErrGameListNotFound,
			wantName: "RPGs",
			wantDesc: nil,
		},
		{
			name:     "unknown ID",
			userID:   1,
			listID:   99,
			details:  GameListDetails{Name: "Mine"},
			wantErr:  ErrGameListNotFound,
			wantName: "RPGs",
			wantDesc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTwoUserStore(t)
			ctx := t.Context()
			if list := createGameList(t, store, 1, "RPGs"); list.ID != 1 {
				t.Fatalf("createGameList(RPGs) ID = %d, want %d", list.ID, 1)
			}
			if list := createGameList(t, store, 1, "Co-op"); list.ID != 2 {
				t.Fatalf("createGameList(Co-op) ID = %d, want %d", list.ID, 2)
			}

			err := store.UpdateGameList(ctx, tt.userID, tt.listID, tt.details)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("UpdateGameList() error = %v, want %v", err, tt.wantErr)
			}

			got, err := store.GetGameList(ctx, 1, 1)
			if err != nil {
				t.Fatalf("GetGameList() error = %v", err)
			}
			if got.Name != tt.wantName {
				t.Errorf("GetGameList() Name = %q, want %q", got.Name, tt.wantName)
			}
			if desc := derefOrNil(got.Description); desc != tt.wantDesc {
				t.Errorf("GetGameList() Description = %v, want %v", desc, tt.wantDesc)
			}
		})
	}
}

// TestDeleteGameList checks that a delete removes the list, that a repeated delete counts as success, and that a
// delete never removes a list of another user.
func TestDeleteGameList(t *testing.T) {
	store := newTwoUserStore(t)
	ctx := t.Context()
	rpgs := createGameList(t, store, 1, "RPGs")
	mine := createGameList(t, store, 2, "Mine")

	if err := store.DeleteGameList(ctx, 1, rpgs.ID); err != nil {
		t.Fatalf("DeleteGameList(rpgs) error = %v", err)
	}
	if _, err := store.GetGameList(ctx, 1, rpgs.ID); !errors.Is(err, ErrGameListNotFound) {
		t.Errorf("GetGameList(rpgs) error = %v, want %v", err, ErrGameListNotFound)
	}

	if err := store.DeleteGameList(ctx, 1, rpgs.ID); err != nil {
		t.Errorf("DeleteGameList(rpgs) again error = %v, want nil", err)
	}

	if err := store.DeleteGameList(ctx, 1, mine.ID); err != nil {
		t.Errorf("DeleteGameList(mine) error = %v, want nil", err)
	}
	if _, err := store.GetGameList(ctx, 2, mine.ID); err != nil {
		t.Errorf("GetGameList(mine) error = %v, want nil", err)
	}
}
