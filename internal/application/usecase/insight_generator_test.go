package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubTextGenerator struct {
	output string
	err    error
	prompt string
}

func (s *stubTextGenerator) Generate(_ context.Context, prompt string) (string, error) {
	s.prompt = prompt
	return s.output, s.err
}

func TestPromptInsightGenerator_Generate(t *testing.T) {
	client := &stubTextGenerator{
		output: `{"summary":"short","tags":["go","rss"]}`,
	}
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
	if !strings.Contains(client.prompt, "Article JSON:") {
		t.Fatalf("prompt missing article payload: %q", client.prompt)
	}
	if !strings.Contains(client.prompt, "summary: in Japanese (ja-JP), readable in about 3 minutes") {
		t.Fatalf("prompt missing summary duration rule: %q", client.prompt)
	}
	if !strings.Contains(client.prompt, "tags: 3 to 8 short tags in English") {
		t.Fatalf("prompt missing tags language rule: %q", client.prompt)
	}
}

func TestPromptInsightGenerator_ParseOutputWithNoise(t *testing.T) {
	client := &stubTextGenerator{
		output: "warning line\n{\"summary\":\"ok\",\"tags\":[\"a\",\"b\"]}\n",
	}
	generator := NewPromptInsightGenerator(client)

	got, err := generator.Generate(context.Background(), InsightRequest{Title: "x"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got.Summary != "ok" {
		t.Fatalf("summary = %q, want %q", got.Summary, "ok")
	}
}

func TestPromptInsightGenerator_GenerateErrors(t *testing.T) {
	tests := []struct {
		name     string
		noClient bool
		client   *stubTextGenerator
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
			generator := NewPromptInsightGenerator(TextGenerator(tt.client))
			if tt.noClient {
				generator = NewPromptInsightGenerator(nil)
			}
			_, err := generator.Generate(context.Background(), InsightRequest{Title: "x"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
