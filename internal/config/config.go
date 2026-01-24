// Package config handles configuration loading and saving.
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"gopkg.in/yaml.v3"
)

// KeyMapConfig defines the configuration for keybindings.
type KeyMapConfig struct {
	Up         string `yaml:"up" kong:"help='Up key',default='k'"`
	Down       string `yaml:"down" kong:"help='Down key',default='j'"`
	Left       string `yaml:"left" kong:"help='Left/Back key',default='h'"`
	Right      string `yaml:"right" kong:"help='Right/Enter key',default='l'"`
	UpPage     string `yaml:"up_page" kong:"help='Page Up key',default='ctrl+u'"`
	DownPage   string `yaml:"down_page" kong:"help='Page Down key',default='ctrl+d'"`
	Top        string `yaml:"top" kong:"help='Top key',default='g'"`
	Bottom     string `yaml:"bottom" kong:"help='Bottom key',default='G'"`
	Open       string `yaml:"open" kong:"help='Open key',default='enter'"`
	Back       string `yaml:"back" kong:"help='Back key',default='esc'"`
	Quit       string `yaml:"quit" kong:"help='Quit key',default='q'"`
	AddFeed    string `yaml:"add_feed" kong:"help='Add feed key',default='a'"`
	DeleteFeed string `yaml:"delete_feed" kong:"help='Delete feed key',default='x'"`
	Refresh    string `yaml:"refresh" kong:"help='Refresh key',default='r'"`
}

// ThemeConfig defines the color theme configuration.
type ThemeConfig struct {
	FeedName string `yaml:"feed_name" kong:"help='Feed name color',default='244'"`
}

// Config represents the application configuration.
type Config struct {
	Feeds       []string     `yaml:"feeds" kong:"help='RSS Feed URLs',default='https://news.ycombinator.com/rss'"`
	KeyMap      KeyMapConfig `yaml:"keymap" kong:"embed,prefix='keymap.'"`
	Theme       ThemeConfig  `yaml:"theme" kong:"embed,prefix='theme.'"`
	HistoryFile string       `yaml:"history_file" kong:"help='History file path'"`

	// Internal
	configPath string `yaml:"-" kong:"-"`
}

// LoadConfig loads the configuration from the specified path or default location.
func LoadConfig(customPath ...string) (*Config, error) {
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

	var cfg Config
	cfg.configPath = configPath

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

	// Save defaults if new file
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	// Set default history path if empty
	if cfg.HistoryFile == "" {
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			home, err := os.UserHomeDir()
			if err == nil {
				dataHome = filepath.Join(home, ".local", "share")
			}
		}
		cfg.HistoryFile = filepath.Join(dataHome, "reazy", "history.jsonl")
	}

	return &cfg, nil
}

func yamlKongLoader(r io.Reader) (kong.Resolver, error) {
	values := map[string]interface{}{}
	if err := yaml.NewDecoder(r).Decode(&values); err != nil {
		if err == io.EOF {
			return nil, nil // Return nil resolver (no op)
		}
		return nil, err
	}

	var f kong.ResolverFunc = func(_ *kong.Context, _ *kong.Path, flag *kong.Flag) (interface{}, error) {
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
						if nextMap, ok := curr[part].(map[string]interface{}); ok {
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

// AddFeed adds a new feed URL to the configuration and saves it.
func (c *Config) AddFeed(url string) error {
	c.Feeds = append(c.Feeds, url)
	return c.Save()
}

// RemoveFeed removes a feed by index and saves the configuration.
func (c *Config) RemoveFeed(index int) error {
	if index < 0 || index >= len(c.Feeds) {
		return fmt.Errorf("invalid feed index: %d", index)
	}
	c.Feeds = append(c.Feeds[:index], c.Feeds[index+1:]...)
	return c.Save()
}

// Save writes the current configuration to the config file.
func (c *Config) Save() error {
	f, err := os.Create(c.configPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return yaml.NewEncoder(f).Encode(c)
}
