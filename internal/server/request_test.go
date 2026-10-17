package server

import (
	"net/http"
	"net/http/httptest"
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
