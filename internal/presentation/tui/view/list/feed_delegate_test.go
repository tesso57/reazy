package listview

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFeedItem satisfies the FeedItem interface.
type testFeedItem struct {
	title string
	url   string
}

func (m testFeedItem) Title() string       { return m.title }
func (m testFeedItem) Description() string { return "" }
func (m testFeedItem) FilterValue() string { return m.title }
func (m testFeedItem) URL() string         { return m.url }

func TestNewFeedDelegate(t *testing.T) {
	d := NewFeedDelegate(lipgloss.Color("205"))
	require.NotNil(t, d)
	assert.Equal(t, 1, d.Height())
	assert.Equal(t, 0, d.Spacing())
}

func TestFeedDelegate_Update(t *testing.T) {
	d := NewFeedDelegate(lipgloss.Color("205"))
	cmd := d.Update(nil, nil)
	assert.Nil(t, cmd)
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
			item:     testFeedItem{title: "Tech News", url: "http://example.com"},
			index:    0,
			mdlIndex: 1, // Not selected
			contains: "Tech News",
		},
		{
			name:     "Selected Feed",
			item:     testFeedItem{title: "Selected Feed", url: "http://example.com/sel"},
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
				assert.Zero(t, buf.Len())
			} else {
				assert.Contains(t, buf.String(), tc.contains)
			}
		})
	}
}
