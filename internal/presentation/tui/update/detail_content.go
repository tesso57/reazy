package update

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
)

const detailSectionDivider = "----------------------------------------"

func buildDetailContent(i *presenter.Item, showAISummary bool) string {
	return buildDetailContentForWidth(i, showAISummary, 0)
}

func buildDetailContentForWidth(i *presenter.Item, showAISummary bool, width int) string {
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
	}
	body := strings.TrimSpace(i.Content)
	if body == "" {
		body = strings.TrimSpace(i.Desc)
	}

	if summary == "" {
		summary = "(No AI summary available.)"
	}
	if body == "" {
		if !i.BodyHydrated {
			body = "(Loading article body...)"
		} else {
			body = "(No article body available. Open it in the browser.)"
		}
	}
	summary = wrapDetailText(summary, width)
	body = wrapDetailText(body, width)

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

func wrapDetailText(text string, width int) string {
	if width <= 0 {
		return text
	}
	// Preserve all text by hard-wrapping long lines (including CJK/no-space text).
	return ansi.Hardwrap(text, width, true)
}
