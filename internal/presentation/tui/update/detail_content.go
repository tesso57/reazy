package update

import (
	"fmt"
	"strings"

	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
)

const detailSectionDivider = "----------------------------------------"

func buildDetailContent(i *presenter.Item, showAISummary bool) string {
	if i == nil {
		return ""
	}

	title := strings.TrimSpace(i.TitleText)
	summaryHeader := "AI Summary"
	if !i.AIUpdatedAt.IsZero() {
		summaryHeader = fmt.Sprintf("AI Summary (%s)", i.AIUpdatedAt.Format("2006-01-02 15:04"))
	}

	summary := strings.TrimSpace(i.AISummary)
	if summary != "" {
		if !showAISummary {
			summary = "(hidden; press Shift+S to toggle)"
		} else if len(i.AITags) > 0 {
			summary = fmt.Sprintf("%s\nAI Tags: %s", summary, strings.Join(i.AITags, ", "))
		}
	} else {
		summary = strings.TrimSpace(i.Desc)
	}
	body := strings.TrimSpace(i.Content)

	if summary == "" {
		summary = "(No AI summary available.)"
	}
	if body == "" {
		body = "(No article body available. Open it in the browser.)"
	}

	if title == "" {
		return fmt.Sprintf(
			"%s\n%s\n%s\n\n%s\nArticle Body\n%s",
			detailSectionDivider, summaryHeader, summary,
			detailSectionDivider, body,
		)
	}

	return fmt.Sprintf(
		"%s\n\n%s\n%s\n%s\n\n%s\nArticle Body\n%s",
		title,
		detailSectionDivider, summaryHeader, summary,
		detailSectionDivider, body,
	)
}
