package update

import (
	"fmt"
	"strings"

	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
)

const detailSectionDivider = "----------------------------------------"

func buildDetailContent(i *presenter.Item) string {
	if i == nil {
		return ""
	}

	title := strings.TrimSpace(i.TitleText)
	summary := strings.TrimSpace(i.Desc)
	body := strings.TrimSpace(i.Content)

	if summary == "" {
		summary = "(No AI summary available.)"
	}
	if body == "" {
		body = "(No article body available. Open it in the browser.)"
	}

	if title == "" {
		return fmt.Sprintf(
			"%s\nAI Summary\n%s\n\n%s\nArticle Body\n%s",
			detailSectionDivider, summary,
			detailSectionDivider, body,
		)
	}

	return fmt.Sprintf(
		"%s\n\n%s\nAI Summary\n%s\n\n%s\nArticle Body\n%s",
		title,
		detailSectionDivider, summary,
		detailSectionDivider, body,
	)
}
