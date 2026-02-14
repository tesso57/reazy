// Package usecase contains application-level services.
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

const (
	maxNewsDigestArticles         = 60
	maxNewsDigestDescriptionChars = 1200
	maxNewsDigestContentChars     = 4000
)

// NewsDigestArticle is one source article for daily news generation.
type NewsDigestArticle struct {
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	FeedTitle   string `json:"feed_title"`
	Published   string `json:"published"`
	Link        string `json:"link"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// NewsDigestRequest is the input payload for daily news generation.
type NewsDigestRequest struct {
	DateKey  string              `json:"date_key"`
	Articles []NewsDigestArticle `json:"articles"`
}

// NewsDigestTopic is one generated topic in the daily news digest.
type NewsDigestTopic struct {
	Title        string
	Summary      string
	Tags         []string
	ArticleGUIDs []string
}

// NewsDigestGenerator abstracts generation of daily news topics.
type NewsDigestGenerator interface {
	Generate(ctx context.Context, req NewsDigestRequest) ([]NewsDigestTopic, error)
}

// DailyNewsDigest is a generated (or cached) daily news payload.
type DailyNewsDigest struct {
	DateKey   string
	Items     []*reading.HistoryItem
	UsedCache bool
}

// NewsDigestService coordinates daily news generation and cache usage.
type NewsDigestService struct {
	Generator NewsDigestGenerator
	Now       func() time.Time
	Location  func() *time.Location
}

// NewNewsDigestService constructs a NewsDigestService.
func NewNewsDigestService(generator NewsDigestGenerator, now func() time.Time, location func() *time.Location) *NewsDigestService {
	return new(NewsDigestService{
		Generator: generator,
		Now:       now,
		Location:  location,
	})
}

// Enabled reports whether daily digest generation is available.
func (s *NewsDigestService) Enabled() bool {
	return s != nil && s.Generator != nil
}

// BuildDaily builds today's digest from history, using cache unless force is true.
func (s *NewsDigestService) BuildDaily(ctx context.Context, history *reading.History, feeds []string, force bool) (DailyNewsDigest, error) {
	if history == nil {
		return DailyNewsDigest{}, errors.New("history is nil")
	}

	dateKey := s.todayDateKey()
	if !force {
		cached := history.DigestItemsByDate(dateKey)
		if len(cached) > 0 {
			return DailyNewsDigest{
				DateKey:   dateKey,
				Items:     cloneHistoryItems(cached),
				UsedCache: true,
			}, nil
		}
	}

	if !s.Enabled() {
		return DailyNewsDigest{}, errors.New("codex integration is disabled")
	}

	articles := history.TodayArticleItems(dateKey, feeds, s.location())
	if len(articles) == 0 {
		return DailyNewsDigest{}, errors.New("no articles available for today's news")
	}
	if len(articles) > maxNewsDigestArticles {
		articles = articles[:maxNewsDigestArticles]
	}

	req := buildNewsDigestRequest(dateKey, articles)
	topics, err := s.Generator.Generate(ctx, req)
	if err != nil {
		return DailyNewsDigest{}, err
	}

	normalized := normalizeNewsDigestTopics(topics, req.Articles)
	if len(normalized) == 0 {
		return DailyNewsDigest{}, errors.New("daily news generation returned no valid topics")
	}

	return DailyNewsDigest{
		DateKey:   dateKey,
		Items:     buildDigestHistoryItems(dateKey, normalized, s.now(), s.location()),
		UsedCache: false,
	}, nil
}

func (s *NewsDigestService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *NewsDigestService) location() *time.Location {
	if s != nil && s.Location != nil {
		if loc := s.Location(); loc != nil {
			return loc
		}
	}
	return time.Local
}

func (s *NewsDigestService) todayDateKey() string {
	return s.now().In(s.location()).Format("2006-01-02")
}

// TodayDateKey returns the date key used for digest generation.
func (s *NewsDigestService) TodayDateKey() string {
	return s.todayDateKey()
}

func buildNewsDigestRequest(dateKey string, articles []*reading.HistoryItem) NewsDigestRequest {
	result := NewsDigestRequest{
		DateKey:  dateKey,
		Articles: make([]NewsDigestArticle, 0, len(articles)),
	}
	for _, article := range articles {
		if article == nil || strings.TrimSpace(article.GUID) == "" {
			continue
		}
		result.Articles = append(result.Articles, NewsDigestArticle{
			GUID:        article.GUID,
			Title:       strings.TrimSpace(article.Title),
			FeedTitle:   strings.TrimSpace(article.FeedTitle),
			Published:   strings.TrimSpace(article.Published),
			Link:        strings.TrimSpace(article.Link),
			Description: limitInsightText(strings.TrimSpace(article.Description), maxNewsDigestDescriptionChars),
			Content:     limitInsightText(strings.TrimSpace(article.Content), maxNewsDigestContentChars),
		})
	}
	return result
}

func normalizeNewsDigestTopics(topics []NewsDigestTopic, source []NewsDigestArticle) []NewsDigestTopic {
	if len(topics) == 0 {
		return nil
	}

	validGUIDs := make(map[string]struct{}, len(source))
	for _, article := range source {
		if strings.TrimSpace(article.GUID) == "" {
			continue
		}
		validGUIDs[article.GUID] = struct{}{}
	}

	normalized := make([]NewsDigestTopic, 0, len(topics))
	for _, topic := range topics {
		title := strings.TrimSpace(topic.Title)
		summary := strings.TrimSpace(topic.Summary)
		if title == "" || summary == "" {
			continue
		}

		guids := make([]string, 0, len(topic.ArticleGUIDs))
		seen := make(map[string]struct{}, len(topic.ArticleGUIDs))
		for _, guid := range topic.ArticleGUIDs {
			guid = strings.TrimSpace(guid)
			if guid == "" {
				continue
			}
			if _, ok := validGUIDs[guid]; !ok {
				continue
			}
			if _, dup := seen[guid]; dup {
				continue
			}
			seen[guid] = struct{}{}
			guids = append(guids, guid)
		}
		if len(guids) == 0 {
			continue
		}

		normalized = append(normalized, NewsDigestTopic{
			Title:        title,
			Summary:      summary,
			Tags:         normalizeTags(topic.Tags),
			ArticleGUIDs: guids,
		})
	}

	return normalized
}

func buildDigestHistoryItems(dateKey string, topics []NewsDigestTopic, savedAt time.Time, loc *time.Location) []*reading.HistoryItem {
	if len(topics) == 0 {
		return nil
	}
	if loc == nil {
		loc = time.Local
	}
	digestDate, err := time.ParseInLocation("2006-01-02", dateKey, loc)
	if err != nil {
		digestDate = savedAt.In(loc)
	}

	items := make([]*reading.HistoryItem, 0, len(topics))
	for index, topic := range topics {
		items = append(items, &reading.HistoryItem{
			GUID:         fmt.Sprintf("%s:%s:%d", reading.NewsDigestKind, dateKey, index+1),
			Kind:         reading.NewsDigestKind,
			Title:        topic.Title,
			Description:  topic.Summary,
			Content:      topic.Summary,
			Published:    dateKey,
			Date:         digestDate.Add(time.Duration(index) * time.Second),
			FeedTitle:    "Daily News",
			FeedURL:      reading.NewsURL,
			SavedAt:      savedAt,
			DigestDate:   dateKey,
			AITags:       append([]string(nil), topic.Tags...),
			RelatedGUIDs: append([]string(nil), topic.ArticleGUIDs...),
		})
	}
	return items
}

func cloneHistoryItems(items []*reading.HistoryItem) []*reading.HistoryItem {
	cloned := make([]*reading.HistoryItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		copyItem := *item
		copyItem.AITags = slices.Clone(item.AITags)
		copyItem.RelatedGUIDs = slices.Clone(item.RelatedGUIDs)
		cloned = append(cloned, &copyItem)
	}
	return cloned
}

// PromptNewsDigestGenerator builds prompts and parses daily digest JSON output.
type PromptNewsDigestGenerator struct {
	Client TextGenerator
}

// NewPromptNewsDigestGenerator constructs a PromptNewsDigestGenerator.
func NewPromptNewsDigestGenerator(client TextGenerator) PromptNewsDigestGenerator {
	return PromptNewsDigestGenerator{Client: client}
}

// Generate implements NewsDigestGenerator.
func (g PromptNewsDigestGenerator) Generate(ctx context.Context, req NewsDigestRequest) ([]NewsDigestTopic, error) {
	if g.Client == nil {
		return nil, errors.New("ai client is not configured")
	}
	raw, err := g.Client.Generate(ctx, buildNewsDigestPrompt(req))
	if err != nil {
		return nil, err
	}
	return parseNewsDigestOutput(raw)
}

func buildNewsDigestPrompt(req NewsDigestRequest) string {
	payload := struct {
		DateKey  string              `json:"date_key"`
		Articles []NewsDigestArticle `json:"articles"`
	}{
		DateKey:  strings.TrimSpace(req.DateKey),
		Articles: make([]NewsDigestArticle, 0, len(req.Articles)),
	}
	for _, article := range req.Articles {
		payload.Articles = append(payload.Articles, NewsDigestArticle{
			GUID:        strings.TrimSpace(article.GUID),
			Title:       strings.TrimSpace(article.Title),
			FeedTitle:   strings.TrimSpace(article.FeedTitle),
			Published:   strings.TrimSpace(article.Published),
			Link:        strings.TrimSpace(article.Link),
			Description: limitInsightText(strings.TrimSpace(article.Description), maxNewsDigestDescriptionChars),
			Content:     limitInsightText(strings.TrimSpace(article.Content), maxNewsDigestContentChars),
		})
	}
	data, _ := json.Marshal(payload)

	return strings.Join([]string{
		"You are helping an RSS reader create a daily news digest.",
		"Group today's articles into coherent topics and summarize each topic.",
		`Return ONLY valid JSON without markdown: {"topics":[{"title":"...","summary":"...","tags":["..."],"article_guids":["..."]}]}`,
		"Rules:",
		"- summary: Japanese (ja-JP), concise and factual.",
		"- tags: short English tags, 2 to 8 items, no duplicates.",
		"- article_guids: must reference only provided GUIDs.",
		"- ignore malformed entries and produce the best possible result.",
		"Input JSON:",
		string(data),
	}, "\n")
}

func parseNewsDigestOutput(raw string) ([]NewsDigestTopic, error) {
	type topicPayload struct {
		Title        string   `json:"title"`
		Summary      string   `json:"summary"`
		Tags         []string `json:"tags"`
		ArticleGUIDs []string `json:"article_guids"`
	}
	type payload struct {
		Topics []topicPayload `json:"topics"`
	}

	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, errors.New("ai client returned empty output")
	}

	tryDecode := func(data string) ([]NewsDigestTopic, error) {
		var out payload
		if err := json.Unmarshal([]byte(data), &out); err != nil {
			return nil, err
		}
		result := make([]NewsDigestTopic, 0, len(out.Topics))
		for _, topic := range out.Topics {
			result = append(result, NewsDigestTopic{
				Title:        topic.Title,
				Summary:      topic.Summary,
				Tags:         topic.Tags,
				ArticleGUIDs: topic.ArticleGUIDs,
			})
		}
		return result, nil
	}

	topics, err := tryDecode(text)
	if err == nil {
		return topics, nil
	}
	jsonObject := extractInsightJSONObject(text)
	if jsonObject == "" {
		return nil, fmt.Errorf("failed to parse ai output as JSON: %w", err)
	}
	topics, decodeErr := tryDecode(jsonObject)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to parse ai output as JSON: %w", decodeErr)
	}
	return topics, nil
}
