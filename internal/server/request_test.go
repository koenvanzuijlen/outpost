package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestPathID(t *testing.T) {
	tests := []struct {
		name string
		ID   int64
		ok   bool
	}{
		{"valid ID", 123, true},
		{"zero ID", 0, false},
		{"negative ID", -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.SetPathValue("id", strconv.FormatInt(tt.ID, 10)) // You would need to set up the request with the URL path and parameters
			got, ok := pathID(r, "id")
			if tt.ok != ok {
				t.Errorf("pathID() = %v, %v; want %v, %v", got, ok, tt.ID, tt.ok)
			}
			if tt.ok && got != tt.ID {
				t.Errorf("pathID() = %v, %v; want %v, %v", got, ok, tt.ID, tt.ok)
			}
			if !tt.ok && got != 0 {
				t.Errorf("pathID() = %v, %v; want %v, %v", got, ok, tt.ID, tt.ok)
			}
		})
	}
}
