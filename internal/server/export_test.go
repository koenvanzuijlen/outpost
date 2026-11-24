package server

import (
	"testing"
	"time"

	"github.com/koenvanzuijlen/outpost/internal/database"
	"github.com/koenvanzuijlen/outpost/internal/database/dbgen"
)

// TestExportJSON checks the exact format of the export file: time zone and truncation, null values, empty lists,
// embedded fields, HTML characters and key names.
func TestExportJSON(t *testing.T) {
	exportedAt := time.Date(2026, 11, 15, 9, 50, 1, 500_000_000, time.FixedZone("CET", 3600))
	date := func(year int, month time.Month, day int) *time.Time {
		d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		return &d
	}

	data := database.ExportData{
		Library: []database.ExportGame{
			{
				Game: dbgen.Game{
					IgdbID:     1942,
					Title:      "The Witcher 3: Wild Hunt",
					ReleasedAt: date(2015, time.May, 19),
				},
				UserGame: dbgen.UserGame{
					Status:  new("completed"),
					Rating:  new(int64(9)),
					Notes:   new("Story & side quests"),
					AddedAt: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
				},
				Copies: []database.ExportCopy{
					{
						Platform: dbgen.Platform{IgdbID: 6, Name: "PC (Microsoft Windows)"},
						OwnedDLC: []dbgen.Game{
							{IgdbID: 12503, Title: "The Witcher 3: Wild Hunt - Blood and Wine"},
						},
					},
					{
						Platform: dbgen.Platform{IgdbID: 48, Name: "PlayStation 4"},
						OwnedDLC: nil,
					},
				},
				Playthroughs: []dbgen.ListPlaythroughsForUserIDRow{
					{
						Playthrough: dbgen.Playthrough{
							Outcome: "completed",
							// 2024-01-03 23:00 in UTC. The export converts dates to UTC.
							StartedAt:  new(time.Date(2024, 1, 4, 1, 0, 0, 0, time.FixedZone("CEST", 2*3600))),
							FinishedAt: date(2024, time.March, 1),
						},
						PlatformName:   new("PC (Microsoft Windows)"),
						PlatformIgdbID: new(int64(6)),
					},
				},
			},
			{
				Game: dbgen.Game{
					IgdbID:       12503,
					Title:        "The Witcher 3: Wild Hunt - Blood and Wine",
					ParentIgdbID: new(int64(1942)),
				},
				UserGame: dbgen.UserGame{
					Status:  new("backlog"),
					AddedAt: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				},
				Copies: nil,
				Playthroughs: []dbgen.ListPlaythroughsForUserIDRow{
					{
						Playthrough: dbgen.Playthrough{Name: new("New Game+"), Outcome: "playing"},
					},
				},
			},
		},
		Wishlist: []database.ExportGame{
			{
				Game: dbgen.Game{
					IgdbID:     194662,
					Title:      "Example Wish Game",
					ReleasedAt: date(2027, time.March, 1),
				},
				UserGame: dbgen.UserGame{
					Wishlisted: true,
					AddedAt:    time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	want := `{
  "format_version": 1,
  "outpost_version": "1.1.0",
  "exported_at": "2026-11-15T08:50:01Z",
  "library": [
    {
      "igdb_id": 1942,
      "title": "The Witcher 3: Wild Hunt",
      "released_on": "2015-05-19",
      "parent_igdb_id": null,
      "status": "completed",
      "rating": 9,
      "notes": "Story & side quests",
      "added_at": "2024-01-02T10:00:00Z",
      "copies": [
        {
          "platform": {
            "igdb_id": 6,
            "name": "PC (Microsoft Windows)"
          },
          "owned_dlc": [
            {
              "igdb_id": 12503,
              "title": "The Witcher 3: Wild Hunt - Blood and Wine"
            }
          ]
        },
        {
          "platform": {
            "igdb_id": 48,
            "name": "PlayStation 4"
          },
          "owned_dlc": []
        }
      ],
      "playthroughs": [
        {
          "name": null,
          "platform": {
            "igdb_id": 6,
            "name": "PC (Microsoft Windows)"
          },
          "outcome": "completed",
          "started_on": "2024-01-03",
          "finished_on": "2024-03-01"
        }
      ]
    },
    {
      "igdb_id": 12503,
      "title": "The Witcher 3: Wild Hunt - Blood and Wine",
      "released_on": null,
      "parent_igdb_id": 1942,
      "status": "backlog",
      "rating": null,
      "notes": null,
      "added_at": "2024-02-01T00:00:00Z",
      "copies": [],
      "playthroughs": [
        {
          "name": "New Game+",
          "platform": null,
          "outcome": "playing",
          "started_on": null,
          "finished_on": null
        }
      ]
    }
  ],
  "wishlist": [
    {
      "igdb_id": 194662,
      "title": "Example Wish Game",
      "released_on": "2027-03-01",
      "parent_igdb_id": null,
      "added_at": "2024-05-01T12:00:00Z"
    }
  ]
}
`

	got, err := marshalExportJSON(newExportFile(data, "1.1.0", exportedAt))
	if err != nil {
		t.Fatalf("marshalExportJSON() error = %v", err)
	}
	if string(got) != want {
		t.Errorf("marshalExportJSON() = %s, want %s", got, want)
	}
}

// TestNewExportListsEmpty checks that a user without games gets empty lists, not nil lists, so the export file holds []
// and not null.
func TestNewExportListsEmpty(t *testing.T) {
	library, wishlist := newExportLists(database.ExportData{})

	if library == nil || len(library) != 0 {
		t.Errorf("newExportLists() library = %#v, want empty and non-nil", library)
	}
	if wishlist == nil || len(wishlist) != 0 {
		t.Errorf("newExportLists() wishlist = %#v, want empty and non-nil", wishlist)
	}
}

// TestExportFilename checks that the file name uses the date in the time zone of the export time.
func TestExportFilename(t *testing.T) {
	tests := []struct {
		name       string
		exportedAt time.Time
		want       string
	}{
		{"zone ahead of UTC", time.Date(2026, 11, 15, 0, 30, 0, 0, time.FixedZone("CET", 3600)), "outpost-2026-11-15.json"},
		{"UTC", time.Date(2026, 1, 2, 23, 59, 0, 0, time.UTC), "outpost-2026-01-02.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exportFilename(tt.exportedAt)
			if got != tt.want {
				t.Errorf("exportFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExportAllFilename checks that the file name of the export of all users uses the date in the time zone of the
// export time, not the date in UTC.
func TestExportAllFilename(t *testing.T) {
	cet := time.FixedZone("CET", 3600)
	tests := []struct {
		name       string
		exportedAt time.Time
		want       string
	}{
		{"late in the day", time.Date(2026, 11, 15, 23, 30, 0, 0, cet), "outpost-all-users-2026-11-15.json"},
		{"next day in CET, previous day in UTC", time.Date(2026, 11, 16, 0, 30, 0, 0, cet), "outpost-all-users-2026-11-16.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exportAllFilename(tt.exportedAt)
			if got != tt.want {
				t.Errorf("exportAllFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}
