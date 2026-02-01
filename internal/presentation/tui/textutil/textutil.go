// Package textutil provides small formatting helpers for TUI text.
package textutil

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// SingleLine collapses whitespace into single spaces.
func SingleLine(text string) string {
	if text == "" {
		return ""
	}
	return strings.Join(strings.Fields(text), " ")
}

// Truncate trims a string to the given width with an ellipsis.
func Truncate(text string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(text, width, "...")
}
