package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

type stubInsightGenerator struct {
	insight Insight
	err     error
}

func (s stubInsightGenerator) Generate(_ context.Context, _ InsightRequest) (Insight, error) {
	return s.insight, s.err
}

func TestInsightService_Generate(t *testing.T) {
	tests := []struct {
		name      string
		service   *InsightService
		request   InsightRequest
		wantErr   bool
		wantTags  []string
		wantBrief string
	}{
		{
			name:    "disabled",
			service: NewInsightService(nil, nil),
			request: InsightRequest{Title: "t"},
			wantErr: true,
		},
		{
			name:    "empty input",
			service: NewInsightService(stubInsightGenerator{}, nil),
			request: InsightRequest{},
			wantErr: true,
		},
		{
			name: "generator error",
			service: NewInsightService(stubInsightGenerator{
				err: errors.New("boom"),
			}, nil),
			request: InsightRequest{Title: "t"},
			wantErr: true,
		},
		{
			name: "empty summary returned",
			service: NewInsightService(stubInsightGenerator{
				insight: Insight{Summary: " ", Tags: []string{"go"}},
			}, nil),
			request: InsightRequest{Title: "t"},
			wantErr: true,
		},
		{
			name: "success with tag normalization",
			service: NewInsightService(stubInsightGenerator{
				insight: Insight{
					Summary: " concise summary ",
					Tags:    []string{"Go", " go ", "RSS", ""},
				},
			}, nil),
			request:   InsightRequest{Title: "t"},
			wantErr:   false,
			wantTags:  []string{"Go", "RSS"},
			wantBrief: "concise summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.service.Generate(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Summary != tt.wantBrief {
				t.Fatalf("Summary = %q, want %q", got.Summary, tt.wantBrief)
			}
			if len(got.Tags) != len(tt.wantTags) {
				t.Fatalf("Tags len = %d, want %d", len(got.Tags), len(tt.wantTags))
			}
			for i := range got.Tags {
				if got.Tags[i] != tt.wantTags[i] {
					t.Fatalf("Tags[%d] = %q, want %q", i, got.Tags[i], tt.wantTags[i])
				}
			}
		})
	}
}

func TestInsightService_ApplyToHistory(t *testing.T) {
	now := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	svc := NewInsightService(nil, func() time.Time { return now })

	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"1": {GUID: "1"},
	})

	ok := svc.ApplyToHistory(history, "1", Insight{
		Summary: "s",
		Tags:    []string{"tag1", "tag2"},
	})
	if !ok {
		t.Fatal("ApplyToHistory should return true")
	}

	item, exists := history.Item("1")
	if !exists {
		t.Fatal("item should exist")
	}
	if item.AISummary != "s" {
		t.Fatalf("AISummary = %q, want %q", item.AISummary, "s")
	}
	if !item.AIUpdatedAt.Equal(now) {
		t.Fatalf("AIUpdatedAt = %v, want %v", item.AIUpdatedAt, now)
	}

	if svc.ApplyToHistory(history, "missing", Insight{}) {
		t.Fatal("ApplyToHistory should return false for missing item")
	}
	if svc.ApplyToHistory(nil, "1", Insight{}) {
		t.Fatal("ApplyToHistory should return false for nil history")
	}
}
