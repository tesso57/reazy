package mainview

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	props := Props{
		Width:  100,
		Height: 50,
		Header: "HEADER",
		Body:   "BODY",
	}

	got := Render(props)

	if !strings.Contains(got, "HEADER") {
		t.Error("Missing header")
	}
	if !strings.Contains(got, "BODY") {
		t.Error("Missing body")
	}

	// Verify style via whitespace?
	// lipgloss adds spaces for width/height.
	// We just ensure content is preserved.
}
