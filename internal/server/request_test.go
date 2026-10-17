package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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
			if tt.ok != ok || derefOrNil(tt.want) != derefOrNil(got) {
				t.Errorf("formOptionalPositiveInt() = %v, %v; want %v, %v", derefOrNil(got), ok, derefOrNil(tt.want), tt.ok)
			}
		})
	}
}
