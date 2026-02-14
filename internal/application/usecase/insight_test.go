package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tesso57/reazy/internal/domain/reading"
)

type mockInsightGenerator struct {
	mock.Mock
}

func (m *mockInsightGenerator) Generate(ctx context.Context, req InsightRequest) (Insight, error) {
	args := m.Called(ctx, req)
	insight, _ := args.Get(0).(Insight)
	return insight, args.Error(1)
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
			service: NewInsightService(&mockInsightGenerator{}, nil),
			request: InsightRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.service.Generate(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("generator error", func(t *testing.T) {
		gen := &mockInsightGenerator{}
		svc := NewInsightService(gen, nil)
		gen.On("Generate", mock.Anything, InsightRequest{Title: "t"}).Return(Insight{}, errors.New("boom")).Once()

		_, err := svc.Generate(context.Background(), InsightRequest{Title: "t"})
		if err == nil {
			t.Fatal("expected error")
		}
		gen.AssertExpectations(t)
	})

	t.Run("empty summary returned", func(t *testing.T) {
		gen := &mockInsightGenerator{}
		svc := NewInsightService(gen, nil)
		gen.On("Generate", mock.Anything, InsightRequest{Title: "t"}).Return(Insight{Summary: " ", Tags: []string{"go"}}, nil).Once()

		_, err := svc.Generate(context.Background(), InsightRequest{Title: "t"})
		if err == nil {
			t.Fatal("expected error")
		}
		gen.AssertExpectations(t)
	})

	t.Run("success with tag normalization", func(t *testing.T) {
		gen := &mockInsightGenerator{}
		svc := NewInsightService(gen, nil)
		gen.On("Generate", mock.Anything, InsightRequest{Title: "t"}).Return(Insight{
			Summary: " concise summary ",
			Tags:    []string{"Go", " go ", "RSS", ""},
		}, nil).Once()

		got, err := svc.Generate(context.Background(), InsightRequest{Title: "t"})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if got.Summary != "concise summary" {
			t.Fatalf("Summary = %q, want %q", got.Summary, "concise summary")
		}
		wantTags := []string{"Go", "RSS"}
		if len(got.Tags) != len(wantTags) {
			t.Fatalf("Tags len = %d, want %d", len(got.Tags), len(wantTags))
		}
		for i := range got.Tags {
			if got.Tags[i] != wantTags[i] {
				t.Fatalf("Tags[%d] = %q, want %q", i, got.Tags[i], wantTags[i])
			}
		}
		gen.AssertExpectations(t)
	})
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
