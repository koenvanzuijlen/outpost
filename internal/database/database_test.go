package database

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
	"github.com/pressly/goose/v3"
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

func TestMigrateUpDownUp(t *testing.T) {
	ctx := t.Context()

	// Initial migration with some data
	store := newTestStore(t)
	addOwnedGame(t, store, 100)
	setupLibraryDLCs(t, store)
	_, err := store.StartPlaythrough(ctx, 1, 100, nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("StartPlaythrough() failed: %v", err)
	}

	// Migrate back down to version 0
	migrations, err := migrationFiles()
	if err != nil {
		t.Fatalf("migrationFiles() failed: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, store.db, migrations)
	if err != nil {
		t.Fatalf("goose provider: %v", err)
	}
	_, err = provider.DownTo(ctx, 0)
	if err != nil {
		t.Fatalf("goose DownTo failed: %v", err)
	}

	gotVersion, err := provider.GetDBVersion(ctx)
	if err != nil {
		t.Fatalf("GetDBVersion() failed: %v", err)
	}
	if gotVersion != 0 {
		t.Errorf("version after DownTo(0) = %d, want %d", gotVersion, 0)
	}

	rows, err := store.db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		t.Fatalf("query sqlite_master failed: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}

	if want := []string{"goose_db_version"}; !slices.Equal(tables, want) {
		t.Errorf("tables after DownTo(0) = %v, want %v", tables, want)
	}

	// Migrate back up and see if inserting is again possible
	_, err = provider.Up(ctx)
	if err != nil {
		t.Fatalf("goose Up failed: %v", err)
	}
	sources := provider.ListSources()
	wantVersion := sources[len(sources)-1].Version

	gotVersion, err = provider.GetDBVersion(ctx)
	if err != nil {
		t.Fatalf("GetDBVersion() failed: %v", err)
	}
	if gotVersion != wantVersion {
		t.Errorf("version after Up = %d, want %d", gotVersion, wantVersion)
	}
	addOwnedGame(t, store, 100)
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
			userGameID := getGame(t, store, 100).UserGame.ID

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
