package update

import (
	"strings"
	"testing"

	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
)

func TestBuildDetailContent(t *testing.T) {
	t.Run("with summary and body", func(t *testing.T) {
		got := buildDetailContent(&presenter.Item{
			TitleText: "1. Example",
			Desc:      "This is an AI summary.",
			Content:   "This is the article body.",
		})

		if !strings.Contains(got, "1. Example") {
			t.Error("expected title in detail content")
		}
		if !strings.Contains(got, "AI Summary") {
			t.Error("expected AI Summary section")
		}
		if !strings.Contains(got, "Article Body") {
			t.Error("expected Article Body section")
		}
		if !strings.Contains(got, "This is an AI summary.") {
			t.Error("expected summary text")
		}
		if !strings.Contains(got, "This is the article body.") {
			t.Error("expected body text")
		}
	})

	t.Run("fallback text when summary/body missing", func(t *testing.T) {
		got := buildDetailContent(&presenter.Item{TitleText: "Only Title"})

		if !strings.Contains(got, "(No AI summary available.)") {
			t.Error("expected fallback summary text")
		}
		if !strings.Contains(got, "(No article body available. Open it in the browser.)") {
			t.Error("expected fallback body text")
		}
	})

	t.Run("nil item", func(t *testing.T) {
		if got := buildDetailContent(nil); got != "" {
			t.Errorf("expected empty content for nil item, got %q", got)
		}
	})
}
