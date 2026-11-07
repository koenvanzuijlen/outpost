package server

import (
	"context"
	"io/fs"
	"os"
	"path"
	"slices"
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

// TestIconListMatchesEmbeddedFiles checks that icons.txt lists each embedded SVG file exactly once, and no other names.
func TestIconListMatchesEmbeddedFiles(t *testing.T) {
	data, err := os.ReadFile("icons/icons.txt")
	if err != nil {
		t.Fatalf("os.ReadFile(icons.txt) error = %v, want nil", err)
	}
	listed := strings.Fields(string(data))

	files, err := fs.Glob(iconFiles, "icons/*.svg")
	if err != nil {
		t.Fatalf("fs.Glob() error = %v, want nil", err)
	}
	embedded := make([]string, 0, len(files))
	for _, f := range files {
		embedded = append(embedded, strings.TrimSuffix(path.Base(f), ".svg"))
	}

	slices.Sort(listed)
	slices.Sort(embedded)
	if !slices.Equal(listed, embedded) {
		t.Errorf("icons.txt names = %v, want %v", listed, embedded)
	}
}
