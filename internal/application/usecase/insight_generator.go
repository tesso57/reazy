// Package usecase contains application-level services.
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	maxInsightContentChars     = 12000
	maxInsightDescriptionChars = 2000
)

// TextGenerator abstracts plain prompt -> text completion.
type TextGenerator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// PromptInsightGenerator builds prompts and parses JSON output from a text generator.
type PromptInsightGenerator struct {
	Client TextGenerator
}

// NewPromptInsightGenerator constructs a PromptInsightGenerator.
func NewPromptInsightGenerator(client TextGenerator) PromptInsightGenerator {
	return PromptInsightGenerator{Client: client}
}

// Generate implements InsightGenerator.
func (g PromptInsightGenerator) Generate(ctx context.Context, req InsightRequest) (Insight, error) {
	if g.Client == nil {
		return Insight{}, errors.New("ai client is not configured")
	}
	raw, err := g.Client.Generate(ctx, buildInsightPrompt(req))
	if err != nil {
		return Insight{}, err
	}
	return parseInsightOutput(raw)
}

func buildInsightPrompt(req InsightRequest) string {
	limited := struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Content     string `json:"content"`
		URL         string `json:"url"`
		Published   string `json:"published"`
		FeedTitle   string `json:"feed_title"`
	}{
		Title:       strings.TrimSpace(req.Title),
		Description: limitInsightText(strings.TrimSpace(req.Description), maxInsightDescriptionChars),
		Content:     limitInsightText(strings.TrimSpace(req.Content), maxInsightContentChars),
		URL:         strings.TrimSpace(req.Link),
		Published:   strings.TrimSpace(req.Published),
		FeedTitle:   strings.TrimSpace(req.FeedTitle),
	}

	payload, _ := json.Marshal(limited)

	return strings.Join([]string{
		"You are helping an RSS reader.",
		"Summarize the article and propose relevant topic tags.",
		`Return ONLY valid JSON without markdown: {"summary":"...","tags":["..."]}`,
		"Rules:",
		"- summary: 2 to 4 sentences.",
		"- tags: 3 to 8 short tags, no duplicates.",
		"- if content is sparse, still provide the best possible summary from available fields.",
		"Article JSON:",
		string(payload),
	}, "\n")
}

func limitInsightText(s string, maxChars int) string {
	if maxChars <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars])
}

func parseInsightOutput(raw string) (Insight, error) {
	type payload struct {
		Summary string   `json:"summary"`
		Tags    []string `json:"tags"`
	}

	text := strings.TrimSpace(raw)
	if text == "" {
		return Insight{}, errors.New("ai client returned empty output")
	}

	tryDecode := func(data string) (Insight, error) {
		var out payload
		if err := json.Unmarshal([]byte(data), &out); err != nil {
			return Insight{}, err
		}
		return Insight{
			Summary: out.Summary,
			Tags:    out.Tags,
		}, nil
	}

	insight, err := tryDecode(text)
	if err == nil {
		return insight, nil
	}

	jsonObject := extractInsightJSONObject(text)
	if jsonObject == "" {
		return Insight{}, fmt.Errorf("failed to parse ai output as JSON: %w", err)
	}
	insight, decodeErr := tryDecode(jsonObject)
	if decodeErr != nil {
		return Insight{}, fmt.Errorf("failed to parse ai output as JSON: %w", decodeErr)
	}
	return insight, nil
}

func extractInsightJSONObject(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < 0 || end < start {
		return ""
	}
	return text[start : end+1]
}
