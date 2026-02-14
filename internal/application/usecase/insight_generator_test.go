package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
)

type mockTextGenerator struct {
	mock.Mock
}

func (m *mockTextGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	out, _ := args.Get(0).(string)
	return out, args.Error(1)
}

func TestPromptInsightGenerator_Generate(t *testing.T) {
	client := &mockTextGenerator{}
	client.On("Generate", mock.Anything, mock.AnythingOfType("string")).Return(`{"summary":"short","tags":["go","rss"]}`, nil).Once()
	generator := NewPromptInsightGenerator(client)

	got, err := generator.Generate(context.Background(), InsightRequest{
		Title:     "Example",
		Content:   "Body text",
		Link:      "https://example.com/post",
		FeedTitle: "Example Feed",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got.Summary != "short" {
		t.Fatalf("summary = %q, want %q", got.Summary, "short")
	}
	if len(got.Tags) != 2 || got.Tags[0] != "go" || got.Tags[1] != "rss" {
		t.Fatalf("tags = %#v, want [go rss]", got.Tags)
	}

	calls := client.Calls
	if len(calls) != 1 {
		t.Fatalf("expected one Generate call, got %d", len(calls))
	}
	prompt, _ := calls[0].Arguments.Get(1).(string)
	if !strings.Contains(prompt, "Article JSON:") {
		t.Fatalf("prompt missing article payload: %q", prompt)
	}
	if !strings.Contains(prompt, "summary: in Japanese (ja-JP), readable in about 3 minutes") {
		t.Fatalf("prompt missing summary duration rule: %q", prompt)
	}
	if !strings.Contains(prompt, "tags: 3 to 8 short tags in English") {
		t.Fatalf("prompt missing tags language rule: %q", prompt)
	}
	client.AssertExpectations(t)
}

func TestPromptInsightGenerator_ParseOutputWithNoise(t *testing.T) {
	client := &mockTextGenerator{}
	client.On("Generate", mock.Anything, mock.AnythingOfType("string")).Return("warning line\n{\"summary\":\"ok\",\"tags\":[\"a\",\"b\"]}\n", nil).Once()
	generator := NewPromptInsightGenerator(client)

	got, err := generator.Generate(context.Background(), InsightRequest{Title: "x"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got.Summary != "ok" {
		t.Fatalf("summary = %q, want %q", got.Summary, "ok")
	}
	client.AssertExpectations(t)
}

func TestPromptInsightGenerator_GenerateErrors(t *testing.T) {
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
				generator := NewPromptInsightGenerator(nil)
				_, err := generator.Generate(context.Background(), InsightRequest{Title: "x"})
				if (err != nil) != tt.wantErr {
					t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			client := &mockTextGenerator{}
			client.On("Generate", mock.Anything, mock.AnythingOfType("string")).Return(tt.output, tt.err).Once()
			generator := NewPromptInsightGenerator(client)

			_, err := generator.Generate(context.Background(), InsightRequest{Title: "x"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
			client.AssertExpectations(t)
		})
	}
}
