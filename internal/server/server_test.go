package server

import (
	"bytes"
	"log/slog"
	"testing"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	server, err := New(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return server
}

func TestTemplatesRender(t *testing.T) {
	server := newTestServer(t)

	for name := range server.templates {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := server.templates[name].ExecuteTemplate(&buf, "base.html", struct{}{}); err != nil {
				t.Errorf("render: %v", err)
			}
			if buf.Len() == 0 {
				t.Errorf("empty content %s", name)
			}
		})
	}
}
