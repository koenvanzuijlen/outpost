package server

import (
	"context"
	"strings"
	"testing"
)

// TestIconRendersSVG checks that a known icon renders its SVG markup.
func TestIconRendersSVG(t *testing.T) {
	var sb strings.Builder
	if err := icon("search").Render(context.Background(), &sb); err != nil {
		t.Fatalf("icon(\"search\").Render() error = %v, want nil", err)
	}
	if !strings.Contains(sb.String(), "<svg") {
		t.Errorf("icon(\"search\") output = %q, want it to contain %q", sb.String(), "<svg")
	}
}

// TestIconUnknownName checks that an unknown icon name returns an error.
func TestIconUnknownName(t *testing.T) {
	var sb strings.Builder
	if err := icon("does-not-exist").Render(context.Background(), &sb); err == nil {
		t.Errorf("icon(\"does-not-exist\").Render() error = nil, want an error")
	}
}
