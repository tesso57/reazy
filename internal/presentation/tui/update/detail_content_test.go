package update

import (
	"strings"
	"testing"
	"time"

	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
)

func TestBuildDetailContent(t *testing.T) {
	t.Run("with summary and body", func(t *testing.T) {
		got := buildDetailContent(&presenter.Item{
			TitleText: "1. Example",
			Desc:      "This is an AI summary.",
			Content:   "This is the article body.",
		}, true)

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
		got := buildDetailContent(&presenter.Item{TitleText: "Only Title"}, true)

		if !strings.Contains(got, "(No AI summary available.)") {
			t.Error("expected fallback summary text")
		}
		if !strings.Contains(got, "(No article body available. Open it in the browser.)") {
			t.Error("expected fallback body text")
		}
	})

	t.Run("nil item", func(t *testing.T) {
		if got := buildDetailContent(nil, true); got != "" {
			t.Errorf("expected empty content for nil item, got %q", got)
		}
	})

	t.Run("generated summary can be hidden", func(t *testing.T) {
		got := buildDetailContent(&presenter.Item{
			TitleText:   "1. Example",
			AISummary:   "Generated summary",
			AITags:      []string{"go", "rss"},
			AIUpdatedAt: time.Date(2026, 2, 7, 10, 30, 0, 0, time.UTC),
			Content:     "Body",
		}, false)

		if !strings.Contains(got, "(hidden; press Shift+S to toggle)") {
			t.Error("expected hidden summary hint")
		}
		if strings.Contains(got, "AI Tags:") {
			t.Error("did not expect tags when summary is hidden")
		}
	})

	t.Run("generated summary shows tags when visible", func(t *testing.T) {
		got := buildDetailContent(&presenter.Item{
			TitleText: "1. Example",
			AISummary: "Generated summary",
			AITags:    []string{"go", "rss"},
			Content:   "Body",
		}, true)

		if !strings.Contains(got, "Generated summary") {
			t.Error("expected generated summary")
		}
		if !strings.Contains(got, "AI Tags: go, rss") {
			t.Error("expected tags line")
		}
	})
}
