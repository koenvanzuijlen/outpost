package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestPathID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int64
		ok    bool
	}{
		{"valid ID", "123", 123, true},
		{"zero ID", "0", 0, false},
		{"negative ID", "-2", 0, false},
		{"non-numeric ID", "abc", 0, false},
		{"empty ID", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.SetPathValue("id", tt.value)
			got, ok := pathID(r, "id")
			if tt.ok != ok || tt.want != got {
				t.Errorf("pathID() = %v, %v; want %v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestFormOptionalPositiveInt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  *int64
		ok    bool
	}{
		{"valid positive", "123", new(int64(123)), true},
		{"zero", "0", nil, false},
		{"negative", "-5", nil, false},
		{"non-numeric", "abc", nil, false},
		{"empty", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(url.Values{"id": {tt.value}}.Encode())
			r := httptest.NewRequest(http.MethodPost, "/", body)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			got, ok := formOptionalPositiveInt(r, "id")
			if ok != tt.ok {
				t.Errorf("formOptionalPositiveInt() ok = %v, want %v", ok, tt.ok)
			}
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("formOptionalPositiveInt() = %v, want nil", *got)
			case tt.want != nil && got == nil:
				t.Errorf("formOptionalPositiveInt() = nil, want %v", *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Errorf("formOptionalPositiveInt() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestFormPositiveInts(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		want   []int64
		wantOK bool
	}{
		{"two values", "igdb_id=1&igdb_id=20", []int64{1, 20}, true},
		{"duplicates kept", "igdb_id=1&igdb_id=1", []int64{1, 1}, true},
		{"no values", "", []int64{}, true},
		{"other field", "other=1", []int64{}, true},
		{"not a number", "igdb_id=abc", nil, false},
		{"zero", "igdb_id=0", nil, false},
		{"negative", "igdb_id=-3", nil, false},
		{"empty value", "igdb_id=", nil, false},
		{"one bad value", "igdb_id=1&igdb_id=abc", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			got, ok := formPositiveInts(r, "igdb_id")
			if ok != tt.wantOK {
				t.Errorf("formPositiveInts() ok = %v, want %v", ok, tt.wantOK)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("formPositiveInts() = %v, want %v", got, tt.want)
			}
			if !tt.wantOK && got != nil {
				t.Errorf("formPositiveInts() = %v, want nil", got)
			}
		})
	}
}

func TestFormOptionalDate(t *testing.T) {
	want := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value string
		want  *time.Time
		ok    bool
	}{
		{"valid", "2026-10-24", &want, true},
		{"empty", "", nil, true},
		{"format", "24-10-2026", nil, false},
		{"impossible", "2026-02-30", nil, false},
		{"non-numeric", "abc", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(url.Values{"id": {tt.value}}.Encode())
			r := httptest.NewRequest(http.MethodPost, "/", body)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			got, ok := formOptionalDate(r, "id")
			if ok != tt.ok {
				t.Errorf("formOptionalDate() ok = %v, want %v", ok, tt.ok)
			}
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("formOptionalDate() = %v, want nil", *got)
			case tt.want != nil && got == nil:
				t.Errorf("formOptionalDate() = nil, want %v", *tt.want)
			case tt.want != nil && !got.Equal(*tt.want):
				t.Errorf("formOptionalDate() = %v, want %v", *got, *tt.want)
			}
		})
	}
}
