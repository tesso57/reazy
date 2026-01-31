package listview

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

// MockArticleItem satisfies the ArticleItem interface
type mockArticleItem struct {
	title     string
	isRead    bool
	feedTitle string
}

func (m mockArticleItem) Title() string       { return m.title }
func (m mockArticleItem) Description() string { return "" }
func (m mockArticleItem) FilterValue() string { return m.title }
func (m mockArticleItem) IsRead() bool        { return m.isRead }
func (m mockArticleItem) FeedTitle() string   { return m.feedTitle }

func TestNewArticleDelegate(t *testing.T) {
	d := NewArticleDelegate()
	if d == nil {
		t.Error("NewArticleDelegate returned nil")
	}
	if d.Height() != 1 {
		t.Errorf("Expected Height 1, got %d", d.Height())
	}
	if d.Spacing() != 0 {
		t.Errorf("Expected Spacing 0, got %d", d.Spacing())
	}
}

func TestArticleDelegate_Update(t *testing.T) {
	d := NewArticleDelegate()
	cmd := d.Update(nil, nil)
	if cmd != nil {
		t.Error("Update should return nil")
	}
}

func TestArticleDelegate_Render(t *testing.T) {
	d := NewArticleDelegate()
	// m := list.Model{} // Unused
	// We need to set the index of the model to match or not match
	// However, list.Model internals are complex to mock perfectly without initialization.
	// But Render only checks m.Index().
	// For bubbletea list, item rendering usually happens via the model.
	// Here we manually call Render.

	tests := []struct {
		name     string
		item     list.Item
		index    int
		mdlIndex int
		contains string
		faint    bool
	}{
		{
			name:     "Unread Item",
			item:     mockArticleItem{title: "Unread Article", isRead: false},
			index:    0,
			mdlIndex: 1, // Not selected
			contains: "Unread Article",
		},
		{
			name:     "Read Item",
			item:     mockArticleItem{title: "Read Article", isRead: true},
			index:    0,
			mdlIndex: 1, // Not selected
			contains: "Read Article",
			// Note: We can't easily assert color/style with plain string check,
			// but we can check the content is present.
			// Faint style usually adds ANSI codes.
		},
		{
			name:     "Selected Item",
			item:     mockArticleItem{title: "Selected Article", isRead: false},
			index:    0,
			mdlIndex: 0, // Selected
			contains: "Selected Article",
		},
		{
			name:     "Invalid Item",
			item:     nil, // triggers !ok check
			index:    0,
			mdlIndex: 0,
			contains: "", // Should write nothing
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			// Setup a mock model with index
			// Since we can't easily set m.Index() on a zero value struct and it has no public setter,
			// we have to rely on `index == m.Index()`.
			// The list.Model struct has unexported `cursor`.
			// However, we can use `list.New` to create a model and set the cursor.
			l := list.New([]list.Item{}, d, 10, 10)
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

			// If we wanted to test Faint, we'd check for ANSI codes, but lipgloss output varies by term env.
			// We trust lipgloss works; just checking logic flow is enough for coverage.
		})
	}
}
