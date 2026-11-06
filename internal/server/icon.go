package server

import (
	"context"
	"embed"
	"fmt"
	"io"

	"github.com/a-h/templ"
)

//go:embed icons/*.svg
var iconFiles embed.FS

// icon renders the Lucide icon icons/<name>.svg inline, hidden from screen readers. The icon takes the current text
// color. Returns an error if the icon file does not exist.
func icon(name string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		data, err := iconFiles.ReadFile("icons/" + name + ".svg")
		if err != nil {
			return fmt.Errorf("read icon %q: %w", name, err)
		}

		_, err = fmt.Fprintf(w, "<span class=\"icon\" aria-hidden=\"true\">%s</span>", data)
		if err != nil {
			return fmt.Errorf("write icon %q: %w", name, err)
		}
		return nil
	})
}
