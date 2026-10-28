package database

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

func TestPragmasApplied(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	var fk int
	err := store.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk)
	if err != nil {
		t.Fatalf("PRAGMA foreign_keys query failed: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

var (
	migrationA = &fstest.MapFile{Data: []byte("-- +goose Up\nCREATE TABLE a (id INTEGER PRIMARY KEY);\n")}
	migrationB = &fstest.MapFile{Data: []byte("-- +goose Up\nCREATE TABLE b (id INTEGER PRIMARY KEY);\n")}
)

func openRawDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		t.Fatalf("sql.Open(%q) failed: %v", path, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func runMigrate(t *testing.T, db *sql.DB, fsys fstest.MapFS, path string) {
	t.Helper()
	err := migrate(t.Context(), db, fsys, slog.New(slog.DiscardHandler), path)
	if err != nil {
		t.Fatalf("migrate() failed: %v", err)
	}
}

func backupFiles(t *testing.T, path string) []string {
	t.Helper()
	matches, err := filepath.Glob(path + ".v*.bak")
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	return matches
}

func TestMigrateBackup(t *testing.T) {
	bothFS := fstest.MapFS{"00001_a.sql": migrationA, "00002_b.sql": migrationB}
	onlyAFS := fstest.MapFS{"00001_a.sql": migrationA}

	t.Run("fresh database creates no backup", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.db")
		db := openRawDB(t, path)

		runMigrate(t, db, bothFS, path)

		if got := backupFiles(t, path); len(got) != 0 {
			t.Errorf("backup files = %v, want none", got)
		}
	})

	t.Run("pending migration creates snapshot", func(t *testing.T) {
		ctx := t.Context()
		path := filepath.Join(t.TempDir(), "test.db")
		db := openRawDB(t, path)

		runMigrate(t, db, onlyAFS, path)
		_, err := db.ExecContext(ctx, "INSERT INTO a(id) VALUES (42)")
		if err != nil {
			t.Fatalf("insert into a failed: %v", err)
		}
		runMigrate(t, db, bothFS, path)

		bakPath := path + ".v1.bak"
		if _, err := os.Stat(bakPath); err != nil {
			t.Fatalf("stat %s: %v", bakPath, err)
		}
		bak := openRawDB(t, bakPath)
		var id int
		err = bak.QueryRowContext(ctx, "SELECT id FROM a").Scan(&id)
		if err != nil {
			t.Fatalf("query backup failed: %v", err)
		}
		if id != 42 {
			t.Errorf("backup row id = %d, want 42", id)
		}
	})

	t.Run("existing backup is kept", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.db")
		db := openRawDB(t, path)

		runMigrate(t, db, onlyAFS, path)
		bakPath := path + ".v1.bak"
		err := os.WriteFile(bakPath, nil, 0o600)
		if err != nil {
			t.Fatalf("create %s: %v", bakPath, err)
		}
		runMigrate(t, db, bothFS, path)

		info, err := os.Stat(bakPath)
		if err != nil {
			t.Fatalf("stat %s: %v", bakPath, err)
		}
		if info.Size() != 0 {
			t.Errorf("backup size = %d, want 0 (should not be overwritten)", info.Size())
		}
	})

	t.Run("no pending migrations creates no backup", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.db")
		db := openRawDB(t, path)

		runMigrate(t, db, bothFS, path)
		runMigrate(t, db, bothFS, path)

		if got := backupFiles(t, path); len(got) != 0 {
			t.Errorf("backup files = %v, want none", got)
		}
	})
}

func TestPlaythroughChecks(t *testing.T) {
	day1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		playName   *string
		outcome    string
		startedAt  *time.Time
		finishedAt *time.Time
		wantErr    bool
	}{
		{"valid", new("PT"), "completed", &day1, &day2, false},
		{"nil name and dates", nil, "completed", nil, nil, false},
		{"empty name", new(""), "playing", nil, nil, true},
		{"playing with finished", nil, "playing", &day1, &day2, true},
		{"finished before started", nil, "completed", &day2, &day1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			store := newTestStore(t)
			addOwnedGame(t, store, 100)
			userGameID := getUserGame(t, store, 100).UserGame.ID

			_, err := store.db.ExecContext(ctx,
				"INSERT INTO playthroughs(user_game_id, name, outcome, started_at, finished_at) VALUES (?, ?, ?, ?, ?)",
				userGameID, tt.playName, tt.outcome, tt.startedAt, tt.finishedAt)
			if (err != nil) != tt.wantErr {
				t.Errorf("insert err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTypeRoundTrip(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	fetchedAt := time.Now()
	_, err := store.queries.UpsertGame(ctx, dbgen.UpsertGameParams{
		IgdbID:       1,
		FetchedAt:    fetchedAt,
		ParentIgdbID: nil,
		Title:        "Test Game",
		GameType:     1,
		CoverUrl:     nil,
		ReleasedAt:   nil,
		Summary:      nil,
	})
	if err != nil {
		t.Fatalf("UpsertGame() failed: %v", err)
	}
	game, err := store.queries.GetGameByIGDBID(ctx, 1)
	if err != nil {
		t.Fatalf("GetGameByIGDBID() failed: %v", err)
	}
	t.Run("Datetime_field", func(t *testing.T) {
		if !game.FetchedAt.Equal(fetchedAt) {
			t.Errorf("FetchedAt = %v, want %v", game.FetchedAt, fetchedAt)
		}
	})
	t.Run("Nullable_field", func(t *testing.T) {
		if game.CoverUrl != nil {
			t.Errorf("CoverUrl = %v, want nil", game.CoverUrl)
		}
	})
}
