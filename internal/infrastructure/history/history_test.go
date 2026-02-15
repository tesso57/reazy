package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

func TestManager_UpsertAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.db")
	m := NewManager(path)

	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	items := []*reading.HistoryItem{
		{
			GUID:         "id1",
			Kind:         reading.ArticleKind,
			Title:        "Title 1",
			Description:  "Desc 1",
			Content:      "Body 1",
			FeedURL:      "feed1",
			FeedTitle:    "Feed 1",
			Date:         now,
			SavedAt:      now,
			IsRead:       false,
			IsBookmarked: true,
			AITags:       []string{"go", "rss"},
			RelatedGUIDs: []string{"x", "y"},
		},
	}

	if err := m.Upsert(items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	meta, err := m.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	if len(meta) != 1 {
		t.Fatalf("LoadMetadata len = %d, want 1", len(meta))
	}
	if meta["id1"].BodyHydrated {
		t.Fatal("article metadata should not be hydrated")
	}
	if meta["id1"].Content != "" {
		t.Fatalf("metadata content should be empty, got %q", meta["id1"].Content)
	}

	full, err := m.LoadByGUID("id1")
	if err != nil {
		t.Fatalf("LoadByGUID failed: %v", err)
	}
	if full == nil {
		t.Fatal("LoadByGUID returned nil")
	}
	if !full.BodyHydrated {
		t.Fatal("LoadByGUID should hydrate body")
	}
	if full.Content != "Body 1" {
		t.Fatalf("content = %q, want Body 1", full.Content)
	}
	if len(full.AITags) != 2 || full.AITags[0] != "go" {
		t.Fatalf("AI tags not round-tripped: %#v", full.AITags)
	}
	if len(full.RelatedGUIDs) != 2 || full.RelatedGUIDs[1] != "y" {
		t.Fatalf("RelatedGUIDs not round-tripped: %#v", full.RelatedGUIDs)
	}
}

func TestManager_Setters(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(filepath.Join(tmpDir, "history.db"))

	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	if err := m.Upsert([]*reading.HistoryItem{{GUID: "id1", Kind: reading.ArticleKind, SavedAt: now}}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if err := m.SetRead("id1", true); err != nil {
		t.Fatalf("SetRead failed: %v", err)
	}
	if err := m.SetBookmark("id1", true); err != nil {
		t.Fatalf("SetBookmark failed: %v", err)
	}
	if err := m.SetInsight("id1", "summary", []string{"go"}, now.Add(time.Minute)); err != nil {
		t.Fatalf("SetInsight failed: %v", err)
	}

	item, err := m.LoadByGUID("id1")
	if err != nil {
		t.Fatalf("LoadByGUID failed: %v", err)
	}
	if item == nil {
		t.Fatal("LoadByGUID returned nil")
	}
	if !item.IsRead {
		t.Fatal("IsRead should be true")
	}
	if !item.IsBookmarked {
		t.Fatal("IsBookmarked should be true")
	}
	if item.AISummary != "summary" {
		t.Fatalf("AISummary = %q, want summary", item.AISummary)
	}
}

func TestManager_ReplaceDigestItemsByDate(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(filepath.Join(tmpDir, "history.db"))

	now := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	seed := []*reading.HistoryItem{
		{GUID: "d_old_1", Kind: reading.NewsDigestKind, DigestDate: "2026-02-14", Content: "old", SavedAt: now},
		{GUID: "d_keep", Kind: reading.NewsDigestKind, DigestDate: "2026-02-13", Content: "keep", SavedAt: now},
	}
	if err := m.Upsert(seed); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if err := m.ReplaceDigestItemsByDate("2026-02-14", []*reading.HistoryItem{
		{GUID: "d_new_1", Title: "New 1", Content: "new", SavedAt: now},
	}); err != nil {
		t.Fatalf("ReplaceDigestItemsByDate failed: %v", err)
	}

	meta, err := m.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}
	if _, ok := meta["d_old_1"]; !ok {
		t.Fatal("old digest should be kept")
	}
	if _, ok := meta["d_keep"]; !ok {
		t.Fatal("digest from other date should remain")
	}
	if item, ok := meta["d_new_1"]; !ok {
		t.Fatal("new digest should exist")
	} else if !item.BodyHydrated {
		t.Fatal("digest should be hydrated in metadata")
	}
}

func TestManager_LoadTodayArticles(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(filepath.Join(tmpDir, "history.db"))
	loc := time.FixedZone("JST", 9*60*60)

	items := []*reading.HistoryItem{
		{
			GUID:    "a1",
			Kind:    reading.ArticleKind,
			FeedURL: "feed1",
			Date:    time.Date(2026, 2, 14, 9, 0, 0, 0, loc),
			SavedAt: time.Date(2026, 2, 14, 10, 0, 0, 0, loc),
			Content: "Body1",
		},
		{
			GUID:    "a2",
			Kind:    reading.ArticleKind,
			FeedURL: "feed2",
			Date:    time.Date(2026, 2, 13, 23, 0, 0, 0, time.UTC),
			SavedAt: time.Date(2026, 2, 14, 1, 0, 0, 0, time.UTC),
			Content: "Body2",
		},
		{
			GUID:       "d1",
			Kind:       reading.NewsDigestKind,
			DigestDate: "2026-02-14",
			Content:    "Digest",
		},
	}
	if err := m.Upsert(items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	got, err := m.LoadTodayArticles("2026-02-14", []string{"feed1", "feed2"}, 60, loc)
	if err != nil {
		t.Fatalf("LoadTodayArticles failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("LoadTodayArticles len = %d, want 2", len(got))
	}
	if got[0].GUID != "a1" {
		t.Fatalf("first item = %s, want a1", got[0].GUID)
	}
	if !got[0].BodyHydrated {
		t.Fatal("today article should be hydrated")
	}
}

func TestManager_JsonlPathResolvesToDB(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "history.jsonl")
	m := NewManager(jsonlPath)

	if err := m.Upsert([]*reading.HistoryItem{{GUID: "1", Kind: reading.ArticleKind}}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "history.db")
	if _, err := m.LoadByGUID("1"); err != nil {
		t.Fatalf("LoadByGUID failed: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected sqlite db at %s: %v", dbPath, err)
	}
}
