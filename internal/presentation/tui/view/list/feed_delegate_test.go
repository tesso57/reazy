package listview

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// MockFeedItem satisfies the FeedItem interface
type mockFeedItem struct {
	title string
	url   string
}

func (m mockFeedItem) Title() string       { return m.title }
func (m mockFeedItem) Description() string { return "" }
func (m mockFeedItem) FilterValue() string { return m.title }
func (m mockFeedItem) URL() string         { return m.url }

func TestNewFeedDelegate(t *testing.T) {
	d := NewFeedDelegate(lipgloss.Color("205"))
	if d == nil {
		t.Error("NewFeedDelegate returned nil")
	}
	if d.Height() != 1 {
		t.Errorf("Expected Height 1, got %d", d.Height())
	}
	if d.Spacing() != 0 {
		t.Errorf("Expected Spacing 0, got %d", d.Spacing())
	}
}

func TestFeedDelegate_Update(t *testing.T) {
	d := NewFeedDelegate(lipgloss.Color("205"))
	cmd := d.Update(nil, nil)
	if cmd != nil {
		t.Error("Update should return nil")
	}
}

func TestFeedDelegate_Render(t *testing.T) {
	d := NewFeedDelegate(lipgloss.Color("205"))

	tests := []struct {
		name     string
		item     list.Item
		index    int
		mdlIndex int
		contains string
	}{
		{
			name:     "Normal Feed",
			item:     mockFeedItem{title: "Tech News", url: "http://example.com"},
			index:    0,
			mdlIndex: 1, // Not selected
			contains: "Tech News",
		},
		{
			name:     "Selected Feed",
			item:     mockFeedItem{title: "Selected Feed", url: "http://example.com/sel"},
			index:    0,
			mdlIndex: 0, // Selected
			contains: "Selected Feed",
		},
		{
			name:     "Invalid Item",
			item:     nil,
			index:    0,
			mdlIndex: 0,
			contains: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			l := list.New([]list.Item{}, d, 80, 10)
			l.Select(tc.mdlIndex)

			d.Render(buf, l, tc.index, tc.item)

			if tc.contains == "" {
				if buf.Len() > 0 {
					t.Errorf("Expected empty output, got %q", buf.String())
				}
			} else {
				if !bytes.Contains(buf.Bytes(), []byte(tc.contains)) {
					t.Errorf("Expected output to contain %q, got %q", tc.contains, buf.String())
				}
			}
		})
	}
}
