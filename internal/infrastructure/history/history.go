// Package history manages persistent storage of feed item states.
package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/tesso57/reazy/internal/domain/reading"
)

// Manager handles loading and saving history.
type Manager struct {
	mu      sync.RWMutex
	path    string
	db      *sql.DB
	once    sync.Once
	initErr error
}

// NewManager creates a new history manager.
func NewManager(path string) *Manager {
	return &Manager{path: resolveDBPath(path)}
}

func resolveDBPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return trimmed
	}
	if strings.EqualFold(filepath.Ext(trimmed), ".jsonl") {
		return filepath.Join(filepath.Dir(trimmed), "history.db")
	}
	return trimmed
}

func (m *Manager) dbConn() (*sql.DB, error) {
	m.once.Do(func() {
		dir := filepath.Dir(m.path)
		if err := os.MkdirAll(dir, 0750); err != nil {
			m.initErr = err
			return
		}
		db, err := sql.Open("sqlite", m.path)
		if err != nil {
			m.initErr = err
			return
		}
		if err := initDB(db); err != nil {
			_ = db.Close()
			m.initErr = err
			return
		}
		m.db = db
	})
	if m.initErr != nil {
		return nil, m.initErr
	}
	if m.db == nil {
		return nil, fmt.Errorf("history db is not initialized")
	}
	return m.db, nil
}

func initDB(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA busy_timeout=5000;",
	}
	for _, stmt := range pragmas {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	schema := []string{
		`CREATE TABLE IF NOT EXISTS history_items (
			guid TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			title TEXT,
			description TEXT,
			content TEXT,
			link TEXT,
			published TEXT,
			date TEXT,
			feed_title TEXT,
			feed_url TEXT,
			is_read INTEGER NOT NULL DEFAULT 0,
			saved_at TEXT,
			is_bookmarked INTEGER NOT NULL DEFAULT 0,
			ai_summary TEXT,
			ai_tags TEXT,
			ai_updated_at TEXT,
			digest_date TEXT,
			related_guids TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_history_feed_kind_date ON history_items (feed_url, kind, date DESC, saved_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_history_bookmarked_kind_date ON history_items (is_bookmarked, kind, date DESC, saved_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_history_kind_digest_date ON history_items (kind, digest_date);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// LoadMetadata loads history items for list rendering without article body payloads.
func (m *Manager) LoadMetadata() (map[string]*reading.HistoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	db, err := m.dbConn()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT guid, kind, title, description,
		       CASE WHEN kind = ? THEN content ELSE '' END,
		       link, published, date, feed_title, feed_url,
		       is_read, saved_at, is_bookmarked,
		       ai_summary, ai_tags, ai_updated_at,
		       digest_date, related_guids
		FROM history_items`, reading.NewsDigestKind)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make(map[string]*reading.HistoryItem)
	for rows.Next() {
		item, err := scanHistoryItem(rows)
		if err != nil {
			continue
		}
		if item == nil || item.GUID == "" {
			continue
		}
		if item.Kind != reading.NewsDigestKind {
			item.Content = ""
			item.BodyHydrated = false
		} else {
			item.BodyHydrated = true
		}
		items[item.GUID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// LoadByGUID loads one fully hydrated history item.
func (m *Manager) LoadByGUID(guid string) (*reading.HistoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	guid = strings.TrimSpace(guid)
	if guid == "" {
		return nil, nil
	}

	db, err := m.dbConn()
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(`
		SELECT guid, kind, title, description, content,
		       link, published, date, feed_title, feed_url,
		       is_read, saved_at, is_bookmarked,
		       ai_summary, ai_tags, ai_updated_at,
		       digest_date, related_guids
		FROM history_items WHERE guid = ?`, guid)
	item, err := scanHistoryItem(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if item != nil {
		item.BodyHydrated = true
	}
	return item, nil
}

// Upsert inserts or updates the given items.
func (m *Manager) Upsert(items []*reading.HistoryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, err := m.dbConn()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO history_items (
			guid, kind, title, description, content,
			link, published, date, feed_title, feed_url,
			is_read, saved_at, is_bookmarked,
			ai_summary, ai_tags, ai_updated_at,
			digest_date, related_guids
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(guid) DO UPDATE SET
			kind = excluded.kind,
			title = excluded.title,
			description = excluded.description,
			content = excluded.content,
			link = excluded.link,
			published = excluded.published,
			date = excluded.date,
			feed_title = excluded.feed_title,
			feed_url = excluded.feed_url,
			is_read = excluded.is_read,
			saved_at = excluded.saved_at,
			is_bookmarked = excluded.is_bookmarked,
			ai_summary = excluded.ai_summary,
			ai_tags = excluded.ai_tags,
			ai_updated_at = excluded.ai_updated_at,
			digest_date = excluded.digest_date,
			related_guids = excluded.related_guids`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, item := range items {
		if item == nil || strings.TrimSpace(item.GUID) == "" {
			continue
		}
		if _, err := stmt.Exec(upsertArgs(item)...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SetRead updates read state for one item.
func (m *Manager) SetRead(guid string, isRead bool) error {
	return m.updateBoolField("is_read", guid, isRead)
}

// SetBookmark updates bookmark state for one item.
func (m *Manager) SetBookmark(guid string, isBookmarked bool) error {
	return m.updateBoolField("is_bookmarked", guid, isBookmarked)
}

func (m *Manager) updateBoolField(field, guid string, value bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	guid = strings.TrimSpace(guid)
	if guid == "" {
		return nil
	}
	if field != "is_read" && field != "is_bookmarked" {
		return fmt.Errorf("unsupported bool field: %s", field)
	}

	db, err := m.dbConn()
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("UPDATE history_items SET %s = ? WHERE guid = ?", field), boolToInt(value), guid)
	return err
}

// SetInsight updates AI fields for one item.
func (m *Manager) SetInsight(guid, summary string, tags []string, updatedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	guid = strings.TrimSpace(guid)
	if guid == "" {
		return nil
	}

	db, err := m.dbConn()
	if err != nil {
		return err
	}
	_, err = db.Exec(
		"UPDATE history_items SET ai_summary = ?, ai_tags = ?, ai_updated_at = ? WHERE guid = ?",
		summary,
		marshalStringSlice(tags),
		timeToText(updatedAt),
		guid,
	)
	return err
}

// ReplaceDigestItemsByDate replaces all digest rows for the specified date.
func (m *Manager) ReplaceDigestItemsByDate(dateKey string, items []*reading.HistoryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dateKey = strings.TrimSpace(dateKey)
	if dateKey == "" {
		return nil
	}

	db, err := m.dbConn()
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		"DELETE FROM history_items WHERE kind = ? AND digest_date = ?",
		reading.NewsDigestKind,
		dateKey,
	); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO history_items (
			guid, kind, title, description, content,
			link, published, date, feed_title, feed_url,
			is_read, saved_at, is_bookmarked,
			ai_summary, ai_tags, ai_updated_at,
			digest_date, related_guids
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(guid) DO UPDATE SET
			kind = excluded.kind,
			title = excluded.title,
			description = excluded.description,
			content = excluded.content,
			link = excluded.link,
			published = excluded.published,
			date = excluded.date,
			feed_title = excluded.feed_title,
			feed_url = excluded.feed_url,
			is_read = excluded.is_read,
			saved_at = excluded.saved_at,
			is_bookmarked = excluded.is_bookmarked,
			ai_summary = excluded.ai_summary,
			ai_tags = excluded.ai_tags,
			ai_updated_at = excluded.ai_updated_at,
			digest_date = excluded.digest_date,
			related_guids = excluded.related_guids`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, item := range items {
		if item == nil || strings.TrimSpace(item.GUID) == "" {
			continue
		}
		copyItem := *item
		copyItem.Kind = reading.NewsDigestKind
		copyItem.DigestDate = dateKey
		copyItem.BodyHydrated = true
		if _, err := stmt.Exec(upsertArgs(&copyItem)...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LoadTodayArticles loads today's article rows with full body for digest generation.
func (m *Manager) LoadTodayArticles(dateKey string, feeds []string, limit int, loc *time.Location) ([]*reading.HistoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	db, err := m.dbConn()
	if err != nil {
		return nil, err
	}

	base := strings.Builder{}
	base.WriteString(`
		SELECT guid, kind, title, description, content,
		       link, published, date, feed_title, feed_url,
		       is_read, saved_at, is_bookmarked,
		       ai_summary, ai_tags, ai_updated_at,
		       digest_date, related_guids
		FROM history_items
		WHERE kind != ?`)
	args := make([]any, 0, len(feeds)+2)
	args = append(args, reading.NewsDigestKind)

	filteredFeeds := make([]string, 0, len(feeds))
	for _, feed := range feeds {
		feed = strings.TrimSpace(feed)
		if feed == "" {
			continue
		}
		filteredFeeds = append(filteredFeeds, feed)
	}
	if len(filteredFeeds) > 0 {
		base.WriteString(" AND feed_url IN (")
		for i := range filteredFeeds {
			if i > 0 {
				base.WriteString(",")
			}
			base.WriteString("?")
			args = append(args, filteredFeeds[i])
		}
		base.WriteString(")")
	}
	base.WriteString(" ORDER BY date DESC, saved_at DESC")

	rows, err := db.Query(base.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*reading.HistoryItem, 0)
	for rows.Next() {
		item, err := scanHistoryItem(rows)
		if err != nil || item == nil {
			continue
		}
		item.BodyHydrated = true
		if dateKey != "" && itemDateKey(item, loc) != dateKey {
			continue
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		return itemSortDate(items[i], loc).After(itemSortDate(items[j], loc))
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanHistoryItem(src scanner) (*reading.HistoryItem, error) {
	var (
		guid, kind, title, desc, content, link   string
		published, dateText, feedTitle, feedURL  string
		savedAtText, aiSummary, aiTagsJSON       string
		aiUpdatedAtText, digestDate, relatedJSON string
		isRead, isBookmarked                     int
	)
	if err := src.Scan(
		&guid, &kind, &title, &desc, &content,
		&link, &published, &dateText, &feedTitle, &feedURL,
		&isRead, &savedAtText, &isBookmarked,
		&aiSummary, &aiTagsJSON, &aiUpdatedAtText,
		&digestDate, &relatedJSON,
	); err != nil {
		return nil, err
	}

	item := &reading.HistoryItem{
		GUID:         guid,
		Kind:         kind,
		Title:        title,
		Description:  desc,
		Content:      content,
		Link:         link,
		Published:    published,
		Date:         parseTime(dateText),
		FeedTitle:    feedTitle,
		FeedURL:      feedURL,
		IsRead:       isRead != 0,
		SavedAt:      parseTime(savedAtText),
		IsBookmarked: isBookmarked != 0,
		AISummary:    aiSummary,
		AITags:       unmarshalStringSlice(aiTagsJSON),
		AIUpdatedAt:  parseTime(aiUpdatedAtText),
		DigestDate:   digestDate,
		RelatedGUIDs: unmarshalStringSlice(relatedJSON),
		BodyHydrated: strings.TrimSpace(content) != "",
	}
	if strings.TrimSpace(item.Kind) == "" {
		item.Kind = reading.ArticleKind
	}
	if item.Kind == reading.NewsDigestKind {
		item.BodyHydrated = true
	}
	return item, nil
}

func upsertArgs(item *reading.HistoryItem) []any {
	if item == nil {
		return make([]any, 18)
	}
	kind := strings.TrimSpace(item.Kind)
	if kind == "" {
		kind = reading.ArticleKind
	}
	if kind == reading.NewsDigestKind {
		item.BodyHydrated = true
	}
	return []any{
		item.GUID,
		kind,
		item.Title,
		item.Description,
		item.Content,
		item.Link,
		item.Published,
		timeToText(item.Date),
		item.FeedTitle,
		item.FeedURL,
		boolToInt(item.IsRead),
		timeToText(item.SavedAt),
		boolToInt(item.IsBookmarked),
		item.AISummary,
		marshalStringSlice(item.AITags),
		timeToText(item.AIUpdatedAt),
		item.DigestDate,
		marshalStringSlice(item.RelatedGUIDs),
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func timeToText(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func parseTime(text string) time.Time {
	text = strings.TrimSpace(text)
	if text == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err == nil {
		return parsed
	}
	parsed, err = time.Parse(time.RFC3339, text)
	if err == nil {
		return parsed
	}
	return time.Time{}
}

func marshalStringSlice(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func unmarshalStringSlice(payload string) []string {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return nil
	}
	return out
}

func itemSortDate(item *reading.HistoryItem, loc *time.Location) time.Time {
	if item == nil {
		return time.Time{}
	}
	value := item.Date
	if value.IsZero() {
		value = item.SavedAt
	}
	if value.IsZero() {
		return value
	}
	if loc == nil {
		loc = time.Local
	}
	return value.In(loc)
}

func itemDateKey(item *reading.HistoryItem, loc *time.Location) string {
	date := itemSortDate(item, loc)
	if date.IsZero() {
		return ""
	}
	return date.Format("2006-01-02")
}
