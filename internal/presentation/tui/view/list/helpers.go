package listview

import (
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/tesso57/reazy/internal/presentation/tui/metrics"
	"github.com/tesso57/reazy/internal/presentation/tui/textutil"
)

func withItemPadding(styles list.DefaultItemStyles) list.DefaultItemStyles {
	styles.NormalTitle = styles.NormalTitle.PaddingRight(metrics.ItemRightPadding)
	styles.SelectedTitle = styles.SelectedTitle.PaddingRight(metrics.ItemRightPadding)
	styles.DimmedTitle = styles.DimmedTitle.PaddingRight(metrics.ItemRightPadding)
	styles.NormalDesc = styles.NormalDesc.PaddingRight(metrics.ItemRightPadding)
	styles.SelectedDesc = styles.SelectedDesc.PaddingRight(metrics.ItemRightPadding)
	styles.DimmedDesc = styles.DimmedDesc.PaddingRight(metrics.ItemRightPadding)
	return styles
}

func itemStyle(styles list.DefaultItemStyles, m list.Model, index int) lipgloss.Style {
	if index == m.Index() {
		return styles.SelectedTitle
	}
	return styles.NormalTitle
}

func truncateItemText(m list.Model, style lipgloss.Style, text string) string {
	maxWidth := m.Width() - style.GetHorizontalFrameSize() - metrics.ItemSafetyPadding
	return textutil.Truncate(text, maxWidth)
}

func renderItemText(w io.Writer, style lipgloss.Style, text string) {
	_, _ = io.WriteString(w, style.Render(text))
}
