// Package config handles configuration loading and saving.
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/domain/subscription"
	"gopkg.in/yaml.v3"
)

// Store manages persisted application settings.
type Store struct {
	Settings   settings.Settings
	configPath string
}

// Load loads the configuration from the specified path or default location.
func Load(customPath ...string) (*Store, error) {
	var configPath string
	if len(customPath) > 0 && customPath[0] != "" {
		configPath = customPath[0]
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(home, ".config", "reazy", "config.yaml")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0750); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	cfg := settings.Settings{}
	store := &Store{Settings: cfg, configPath: configPath}

	var options []kong.Option

	// Only add configuration loader if file exists
	if _, err := os.Stat(configPath); err == nil {
		options = append(options, kong.Configuration(yamlKongLoader, configPath))
	}

	parser, err := kong.New(&cfg, options...)
	if err != nil {
		return nil, err
	}

	_, err = parser.Parse([]string{})
	if err != nil {
		return nil, err
	}

	store.Settings = cfg
	if groups, err := loadFeedGroupsFromConfig(configPath); err != nil {
		return nil, err
	} else if groups != nil {
		store.Settings.FeedGroups = groups
	}
	store.Settings.Feeds = normalizeFeeds(store.Settings.Feeds)
	store.Settings.FeedGroups = normalizeFeedGroups(store.Settings.FeedGroups)
	store.Settings.HistoryFile = normalizeHistoryPath(store.Settings.HistoryFile)

	// Set default history path if empty.
	if store.Settings.HistoryFile == "" {
		store.Settings.HistoryFile = filepath.Join(defaultDataHome(), "reazy", "history.db")
	}

	// Save defaults if new file
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := store.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	return store, nil
}

func normalizeFeeds(feeds []string) []string {
	if len(feeds) == 0 {
		return feeds
	}
	normalized := make([]string, 0, len(feeds))
	for _, feed := range feeds {
		for item := range strings.FieldsSeq(feed) {
			if item != "" {
				normalized = append(normalized, item)
			}
		}
	}
	return normalized
}

func normalizeFeedGroups(groups []subscription.FeedGroup) []subscription.FeedGroup {
	if len(groups) == 0 {
		return groups
	}

	normalized := make([]subscription.FeedGroup, 0, len(groups))
	for _, group := range groups {
		name := strings.TrimSpace(group.Name)
		if name == "" {
			continue
		}
		feeds := normalizeFeeds(group.Feeds)
		if len(feeds) == 0 {
			continue
		}
		normalized = append(normalized, subscription.FeedGroup{
			Name:  name,
			Feeds: feeds,
		})
	}
	return normalized
}

func loadFeedGroupsFromConfig(configPath string) ([]subscription.FeedGroup, error) {
	f, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var raw struct {
		FeedGroups []subscription.FeedGroup `yaml:"feed_groups"`
	}
	if err := yaml.NewDecoder(f).Decode(&raw); err != nil && err != io.EOF {
		return nil, err
	}
	return raw.FeedGroups, nil
}

func defaultDataHome() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome != "" {
		return dataHome
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".local", "share")
}

func normalizeHistoryPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.EqualFold(filepath.Ext(path), ".jsonl") {
		return filepath.Join(filepath.Dir(path), "history.db")
	}
	return path
}

func yamlKongLoader(r io.Reader) (kong.Resolver, error) {
	values := map[string]any{}
	if err := yaml.NewDecoder(r).Decode(&values); err != nil {
		if err == io.EOF {
			return nil, nil // Return nil resolver (no op)
		}
		return nil, err
	}

	var f kong.ResolverFunc = func(_ *kong.Context, _ *kong.Path, flag *kong.Flag) (any, error) {
		// Try various naming conventions
		names := []string{flag.Name, strings.ReplaceAll(flag.Name, "-", "_")}
		for _, name := range names {
			// Check direct match
			if v, ok := values[name]; ok {
				return v, nil
			}

			// Check nested dot-notation
			parts := strings.Split(name, ".")
			if len(parts) > 1 {
				curr := values
				for i, part := range parts {
					if i == len(parts)-1 {
						if v, ok := curr[part]; ok {
							return v, nil
						}
					} else {
						if nextMap, ok := curr[part].(map[string]any); ok {
							curr = nextMap
						} else {
							break
						}
					}
				}
			}
		}
		return nil, nil
	}
	return f, nil
}

// List returns the currently configured feed URLs.
func (s *Store) List() ([]string, error) {
	feeds := s.Settings.FlattenedFeeds()
	return feeds, nil
}

// ListGroups returns configured feed groups.
func (s *Store) ListGroups() ([]subscription.FeedGroup, error) {
	if len(s.Settings.FeedGroups) == 0 {
		return nil, nil
	}
	groups := make([]subscription.FeedGroup, 0, len(s.Settings.FeedGroups))
	for _, group := range s.Settings.FeedGroups {
		groups = append(groups, subscription.FeedGroup{
			Name:  group.Name,
			Feeds: append([]string(nil), group.Feeds...),
		})
	}
	return groups, nil
}

// ReplaceFeedGroups replaces grouped and ungrouped feed settings.
func (s *Store) ReplaceFeedGroups(groups []subscription.FeedGroup, ungrouped []string) error {
	s.Settings.FeedGroups = normalizeFeedGroups(groups)
	s.Settings.Feeds = normalizeFeeds(ungrouped)
	return s.Save()
}

// Add appends a new feed URL and saves the configuration.
func (s *Store) Add(url string) error {
	s.Settings.Feeds = append(s.Settings.Feeds, url)
	return s.Save()
}

// Remove deletes a feed by index and saves the configuration.
func (s *Store) Remove(index int) error {
	total := len(s.Settings.FlattenedFeeds())
	if index < 0 || index >= total {
		return fmt.Errorf("invalid feed index: %d", index)
	}

	remaining := index
	for groupIndex := range s.Settings.FeedGroups {
		groupLen := len(s.Settings.FeedGroups[groupIndex].Feeds)
		if remaining < groupLen {
			feeds := s.Settings.FeedGroups[groupIndex].Feeds
			s.Settings.FeedGroups[groupIndex].Feeds = append(feeds[:remaining], feeds[remaining+1:]...)
			if len(s.Settings.FeedGroups[groupIndex].Feeds) == 0 {
				s.Settings.FeedGroups = append(s.Settings.FeedGroups[:groupIndex], s.Settings.FeedGroups[groupIndex+1:]...)
			}
			return s.Save()
		}
		remaining -= groupLen
	}

	s.Settings.Feeds = append(s.Settings.Feeds[:remaining], s.Settings.Feeds[remaining+1:]...)
	return s.Save()
}

// Save writes the current settings to the config file.
func (s *Store) Save() error {
	f, err := os.Create(s.configPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return yaml.NewEncoder(f).Encode(s.Settings)
}
