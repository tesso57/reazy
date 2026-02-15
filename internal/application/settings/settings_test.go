package settings

import (
	"testing"

	"github.com/tesso57/reazy/internal/domain/subscription"
)

func TestSettings_FlattenedFeeds(t *testing.T) {
	cfg := Settings{
		FeedGroups: []subscription.FeedGroup{
			{Name: "Tech", Feeds: []string{"https://example.com/a.xml", "https://example.com/b.xml"}},
		},
		Feeds: []string{"https://example.com/c.xml"},
	}

	got := cfg.FlattenedFeeds()
	want := []string{
		"https://example.com/a.xml",
		"https://example.com/b.xml",
		"https://example.com/c.xml",
	}

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
