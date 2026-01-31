package layout

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	props := Props{
		Sidebar: "SIDEBAR",
		Main:    "MAIN",
		Footer:  "FOOTER",
	}

	got := Render(props)

	// Check containment
	if !strings.Contains(got, "SIDEBAR") {
		t.Error("Missing sidebar content")
	}
	if !strings.Contains(got, "MAIN") {
		t.Error("Missing main content")
	}
	if !strings.Contains(got, "FOOTER") {
		t.Error("Missing footer content")
	}

	// Ideally we check structure (Horizontal/Vertical)
	// But simply ensuring all parts are joined is sufficient for this simple layout function.
}
