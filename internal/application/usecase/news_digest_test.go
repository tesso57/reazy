package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

type stubNewsDigestGenerator struct {
	topics  []NewsDigestTopic
	err     error
	calls   int
	lastReq NewsDigestRequest
}

func (s *stubNewsDigestGenerator) Generate(_ context.Context, req NewsDigestRequest) ([]NewsDigestTopic, error) {
	s.calls++
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	return s.topics, nil
}

func TestNewsDigestService_BuildDaily_UsesCache(t *testing.T) {
	now := time.Date(2026, 2, 14, 9, 0, 0, 0, time.Local)
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"d1": {
			GUID:       "d1",
			Kind:       reading.NewsDigestKind,
			DigestDate: "2026-02-14",
			Title:      "Cached",
		},
	})
	gen := &stubNewsDigestGenerator{}
	svc := NewNewsDigestService(gen, func() time.Time { return now }, nil)

	got, err := svc.BuildDaily(context.Background(), history, []string{"feed1"}, false)
	if err != nil {
		t.Fatalf("BuildDaily() error = %v", err)
	}
	if !got.UsedCache {
		t.Fatal("expected cache to be used")
	}
	if len(got.Items) != 1 || got.Items[0].GUID != "d1" {
		t.Fatalf("unexpected cached items: %#v", got.Items)
	}
	if gen.calls != 0 {
		t.Fatalf("generator calls = %d, want 0", gen.calls)
	}
}

func TestNewsDigestService_BuildDaily_ForceRegenerates(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	now := time.Date(2026, 2, 14, 9, 0, 0, 0, loc)
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"d1": {
			GUID:       "d1",
			Kind:       reading.NewsDigestKind,
			DigestDate: "2026-02-14",
			Title:      "Cached",
		},
		"a1": {
			GUID:      "a1",
			Kind:      reading.ArticleKind,
			Title:     "Article 1",
			FeedURL:   "feed1",
			FeedTitle: "Feed 1",
			Date:      time.Date(2026, 2, 14, 8, 0, 0, 0, loc),
		},
	})
	gen := &stubNewsDigestGenerator{
		topics: []NewsDigestTopic{
			{
				Title:        "Topic",
				Summary:      "Summary",
				Tags:         []string{"go"},
				ArticleGUIDs: []string{"a1"},
			},
		},
	}
	svc := NewNewsDigestService(gen, func() time.Time { return now }, func() *time.Location { return loc })

	got, err := svc.BuildDaily(context.Background(), history, []string{"feed1"}, true)
	if err != nil {
		t.Fatalf("BuildDaily(force) error = %v", err)
	}
	if got.UsedCache {
		t.Fatal("force build should not use cache")
	}
	if gen.calls != 1 {
		t.Fatalf("generator calls = %d, want 1", gen.calls)
	}
	if len(got.Items) != 1 {
		t.Fatalf("generated items len = %d, want 1", len(got.Items))
	}
	if got.Items[0].Kind != reading.NewsDigestKind {
		t.Fatalf("kind = %q, want %q", got.Items[0].Kind, reading.NewsDigestKind)
	}
}

func TestNewsDigestService_BuildDaily_LimitsAndFilters(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	now := time.Date(2026, 2, 14, 9, 0, 0, 0, loc)
	items := make(map[string]*reading.HistoryItem)
	for i := 0; i < 70; i++ {
		guid := "a" + strconv.Itoa(i)
		items[guid] = &reading.HistoryItem{
			GUID:      guid,
			Kind:      reading.ArticleKind,
			Title:     "Article",
			FeedURL:   "feed1",
			FeedTitle: "Feed 1",
			Date:      now.Add(-time.Duration(i) * time.Minute),
		}
	}
	history := reading.NewHistory(items)
	gen := &stubNewsDigestGenerator{
		topics: []NewsDigestTopic{
			{
				Title:        "T1",
				Summary:      "S1",
				Tags:         []string{"go", "go", "rss"},
				ArticleGUIDs: []string{"a0", "unknown"},
			},
			{
				Title:        " ",
				Summary:      "skip",
				ArticleGUIDs: []string{"a1"},
			},
		},
	}
	svc := NewNewsDigestService(gen, func() time.Time { return now }, func() *time.Location { return loc })

	got, err := svc.BuildDaily(context.Background(), history, []string{"feed1"}, true)
	if err != nil {
		t.Fatalf("BuildDaily() error = %v", err)
	}
	if len(gen.lastReq.Articles) != maxNewsDigestArticles {
		t.Fatalf("request articles len = %d, want %d", len(gen.lastReq.Articles), maxNewsDigestArticles)
	}
	if len(got.Items) != 1 {
		t.Fatalf("generated items len = %d, want 1", len(got.Items))
	}
	if len(got.Items[0].RelatedGUIDs) != 1 || got.Items[0].RelatedGUIDs[0] != "a0" {
		t.Fatalf("related guids = %#v, want [a0]", got.Items[0].RelatedGUIDs)
	}
	if len(got.Items[0].AITags) != 2 {
		t.Fatalf("tags should be normalized, got %#v", got.Items[0].AITags)
	}
}

func TestNewsDigestService_BuildDaily_Errors(t *testing.T) {
	now := time.Date(2026, 2, 14, 9, 0, 0, 0, time.Local)
	history := reading.NewHistory(map[string]*reading.HistoryItem{})
	svc := NewNewsDigestService(nil, func() time.Time { return now }, nil)

	_, err := svc.BuildDaily(context.Background(), history, nil, false)
	if err == nil {
		t.Fatal("expected error when generator is disabled and no cache")
	}
}

func TestPromptNewsDigestGenerator_Generate(t *testing.T) {
	client := &stubTextGenerator{
		output: `{"topics":[{"title":"Top","summary":"要約","tags":["go","rss"],"article_guids":["a1","a2"]}]}`,
	}
	gen := NewPromptNewsDigestGenerator(client)

	topics, err := gen.Generate(context.Background(), NewsDigestRequest{
		DateKey: "2026-02-14",
		Articles: []NewsDigestArticle{
			{GUID: "a1", Title: "A1"},
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(topics) != 1 {
		t.Fatalf("topics len = %d, want 1", len(topics))
	}
	if topics[0].Title != "Top" {
		t.Fatalf("title = %q, want Top", topics[0].Title)
	}
	if !strings.Contains(client.prompt, "article_guids") {
		t.Fatalf("prompt should mention article_guids, got: %q", client.prompt)
	}
}

func TestPromptNewsDigestGenerator_ParseWithNoise(t *testing.T) {
	client := &stubTextGenerator{
		output: "warn\n{\"topics\":[{\"title\":\"Top\",\"summary\":\"要約\",\"tags\":[\"go\"],\"article_guids\":[\"a1\"]}]}\n",
	}
	gen := NewPromptNewsDigestGenerator(client)

	topics, err := gen.Generate(context.Background(), NewsDigestRequest{DateKey: "2026-02-14"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(topics) != 1 || topics[0].Title != "Top" {
		t.Fatalf("unexpected topics: %#v", topics)
	}
}

func TestPromptNewsDigestGenerator_GenerateErrors(t *testing.T) {
	tests := []struct {
		name     string
		client   *stubTextGenerator
		noClient bool
		wantErr  bool
	}{
		{
			name:     "no client",
			noClient: true,
			wantErr:  true,
		},
		{
			name: "client error",
			client: &stubTextGenerator{
				err: errors.New("provider error"),
			},
			wantErr: true,
		},
		{
			name: "invalid json",
			client: &stubTextGenerator{
				output: "not json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewPromptNewsDigestGenerator(TextGenerator(tt.client))
			if tt.noClient {
				generator = NewPromptNewsDigestGenerator(nil)
			}
			_, err := generator.Generate(context.Background(), NewsDigestRequest{DateKey: "2026-02-14"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
