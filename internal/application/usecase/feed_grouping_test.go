package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/tesso57/reazy/internal/domain/subscription"
)

type mockFeedGroupingGenerator struct {
	mock.Mock
}

func (m *mockFeedGroupingGenerator) Generate(ctx context.Context, req FeedGroupingRequest) ([]subscription.FeedGroup, error) {
	args := m.Called(ctx, req)
	groups, _ := args.Get(0).([]subscription.FeedGroup)
	return groups, args.Error(1)
}

func TestFeedGroupingService_Group(t *testing.T) {
	gen := &mockFeedGroupingGenerator{}
	gen.On("Generate", mock.Anything, mock.Anything).Return([]subscription.FeedGroup{
		{
			Name:  "Tech",
			Feeds: []string{"https://news.ycombinator.com/rss", "https://github.com/golang/go/releases.atom"},
		},
		{
			Name:  "Tech",
			Feeds: []string{"https://github.com/rust-lang/rust/releases.atom"},
		},
		{
			Name:  "Invalid",
			Feeds: []string{"https://unknown.example.com/rss"},
		},
	}, nil).Once()
	svc := NewFeedGroupingService(gen)

	got, err := svc.Group(context.Background(), []string{
		"https://news.ycombinator.com/rss",
		"https://github.com/golang/go/releases.atom",
		"https://github.com/rust-lang/rust/releases.atom",
		"https://planetpython.org/rss20.xml",
	})
	if err != nil {
		t.Fatalf("Group() error = %v", err)
	}

	if len(got.Groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(got.Groups))
	}
	if got.Groups[0].Name != "Tech" {
		t.Fatalf("group name = %q, want Tech", got.Groups[0].Name)
	}
	if len(got.Groups[0].Feeds) != 3 {
		t.Fatalf("len(group feeds) = %d, want 3", len(got.Groups[0].Feeds))
	}

	if len(got.Ungrouped) != 1 || got.Ungrouped[0] != "https://planetpython.org/rss20.xml" {
		t.Fatalf("ungrouped = %#v, want [planetpython]", got.Ungrouped)
	}
	gen.AssertExpectations(t)
}

func TestFeedGroupingService_GroupErrors(t *testing.T) {
	tests := []struct {
		name    string
		service *FeedGroupingService
		feeds   []string
		wantErr bool
	}{
		{
			name:    "disabled",
			service: NewFeedGroupingService(nil),
			feeds:   []string{"https://a", "https://b"},
			wantErr: true,
		},
		{
			name: "too few feeds",
			service: NewFeedGroupingService(&mockFeedGroupingGenerator{
				Mock: mock.Mock{},
			}),
			feeds:   []string{"https://a"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.service.Group(context.Background(), tt.feeds)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Group() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	gen := &mockFeedGroupingGenerator{}
	gen.On("Generate", mock.Anything, mock.Anything).Return([]subscription.FeedGroup{
		{Name: "Empty", Feeds: []string{"https://unknown.example.com"}},
	}, nil).Once()
	svc := NewFeedGroupingService(gen)
	if _, err := svc.Group(context.Background(), []string{"https://a.example.com/rss", "https://b.example.com/rss"}); err == nil {
		t.Fatal("expected error for invalid AI output")
	}
	gen.AssertExpectations(t)
}

func TestPromptFeedGroupingGenerator_Generate(t *testing.T) {
	client := &mockTextGenerator{}
	client.On("Generate", mock.Anything, mock.AnythingOfType("string")).Return(`{"groups":[{"name":"Tech","feeds":["https://news.ycombinator.com/rss"]}]}`, nil).Once()
	gen := NewPromptFeedGroupingGenerator(client)

	groups, err := gen.Generate(context.Background(), FeedGroupingRequest{
		Feeds: []FeedGroupingFeed{
			{URL: "https://news.ycombinator.com/rss", Host: "news.ycombinator.com"},
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "Tech" {
		t.Fatalf("unexpected groups: %#v", groups)
	}

	calls := client.Calls
	if len(calls) != 1 {
		t.Fatalf("expected one Generate call, got %d", len(calls))
	}
	prompt, _ := calls[0].Arguments.Get(1).(string)
	if !strings.Contains(prompt, `"groups":[{"name":"...","feeds":["..."]}]`) {
		t.Fatalf("prompt missing JSON contract: %q", prompt)
	}
	client.AssertExpectations(t)
}

func TestPromptFeedGroupingGenerator_ParseWithNoise(t *testing.T) {
	client := &mockTextGenerator{}
	client.On("Generate", mock.Anything, mock.AnythingOfType("string")).Return("warn\n{\"groups\":[{\"name\":\"Tech\",\"feeds\":[\"https://news.ycombinator.com/rss\"]}]}\n", nil).Once()
	gen := NewPromptFeedGroupingGenerator(client)

	groups, err := gen.Generate(context.Background(), FeedGroupingRequest{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "Tech" {
		t.Fatalf("unexpected groups: %#v", groups)
	}
	client.AssertExpectations(t)
}

func TestPromptFeedGroupingGenerator_GenerateErrors(t *testing.T) {
	tests := []struct {
		name     string
		noClient bool
		output   string
		err      error
		wantErr  bool
	}{
		{name: "no client", noClient: true, wantErr: true},
		{name: "client error", err: errors.New("provider error"), wantErr: true},
		{name: "invalid json", output: "not json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.noClient {
				gen := NewPromptFeedGroupingGenerator(nil)
				_, err := gen.Generate(context.Background(), FeedGroupingRequest{})
				if (err != nil) != tt.wantErr {
					t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			client := &mockTextGenerator{}
			client.On("Generate", mock.Anything, mock.AnythingOfType("string")).Return(tt.output, tt.err).Once()
			gen := NewPromptFeedGroupingGenerator(client)

			_, err := gen.Generate(context.Background(), FeedGroupingRequest{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			client.AssertExpectations(t)
		})
	}
}
