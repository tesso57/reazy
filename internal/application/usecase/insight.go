// Package usecase contains application-level services.
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

// InsightRequest is the structured input for AI insight generation.
type InsightRequest struct {
	Title       string
	Description string
	Content     string
	Link        string
	Published   string
	FeedTitle   string
}

// Insight is the AI-generated summary and tags.
type Insight struct {
	Summary string
	Tags    []string
}

// InsightGenerator abstracts AI insight generation.
type InsightGenerator interface {
	Generate(ctx context.Context, req InsightRequest) (Insight, error)
}

// InsightService coordinates insight generation and history updates.
type InsightService struct {
	Generator InsightGenerator
	Now       func() time.Time
}

// NewInsightService constructs an InsightService.
func NewInsightService(generator InsightGenerator, now func() time.Time) *InsightService {
	return new(InsightService{
		Generator: generator,
		Now:       now,
	})
}

// Enabled reports whether generation is available.
func (s *InsightService) Enabled() bool {
	return s != nil && s.Generator != nil
}

// Generate runs insight generation for the given request.
func (s *InsightService) Generate(ctx context.Context, req InsightRequest) (Insight, error) {
	if s == nil || s.Generator == nil {
		return Insight{}, errors.New("codex integration is disabled")
	}
	if strings.TrimSpace(req.Title) == "" && strings.TrimSpace(req.Content) == "" && strings.TrimSpace(req.Description) == "" {
		return Insight{}, errors.New("article has no content to summarize")
	}

	insight, err := s.Generator.Generate(ctx, req)
	if err != nil {
		return Insight{}, err
	}

	insight.Summary = strings.TrimSpace(insight.Summary)
	insight.Tags = normalizeTags(insight.Tags)
	if insight.Summary == "" {
		return Insight{}, errors.New("empty summary returned by codex")
	}
	return insight, nil
}

// ApplyToHistory updates one article in history with generated insight.
func (s *InsightService) ApplyToHistory(history *reading.History, guid string, insight Insight) bool {
	if s == nil || history == nil {
		return false
	}
	return history.SetInsight(guid, insight.Summary, insight.Tags, s.now())
}

func (s *InsightService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, t)
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
