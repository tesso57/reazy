// Package settings defines application-level configuration data.
package settings

import "github.com/tesso57/reazy/internal/domain/subscription"

// KeyMapConfig defines the configuration for keybindings.
type KeyMapConfig struct {
	Up            string `yaml:"up" kong:"help='Up key',default='k'"`
	Down          string `yaml:"down" kong:"help='Down key',default='j'"`
	Left          string `yaml:"left" kong:"help='Left/Back key',default='h'"`
	Right         string `yaml:"right" kong:"help='Right/Enter key',default='l'"`
	UpPage        string `yaml:"up_page" kong:"help='Page Up key',default='ctrl+u'"`
	DownPage      string `yaml:"down_page" kong:"help='Page Down key',default='ctrl+d'"`
	Top           string `yaml:"top" kong:"help='Top key',default='g'"`
	Bottom        string `yaml:"bottom" kong:"help='Bottom key',default='G'"`
	Open          string `yaml:"open" kong:"help='Open key',default='enter'"`
	Back          string `yaml:"back" kong:"help='Back key',default='esc'"`
	Quit          string `yaml:"quit" kong:"help='Quit key',default='q'"`
	AddFeed       string `yaml:"add_feed" kong:"help='Add feed key',default='a'"`
	DeleteFeed    string `yaml:"delete_feed" kong:"help='Delete feed key',default='x'"`
	GroupFeeds    string `yaml:"group_feeds" kong:"help='AI group feeds key',default='z'"`
	Refresh       string `yaml:"refresh" kong:"help='Refresh key',default='r'"`
	Bookmark      string `yaml:"bookmark" kong:"help='Bookmark key',default='b'"`
	Summarize     string `yaml:"summarize" kong:"help='Generate AI summary/tags key',default='s'"`
	ToggleSummary string `yaml:"toggle_summary" kong:"help='Toggle AI summary visibility key',default='S'"`
}

// ThemeConfig defines the color theme configuration.
type ThemeConfig struct {
	FeedName string `yaml:"feed_name" kong:"help='Feed name color',default='244'"`
}

// CodexConfig defines Codex CLI integration settings.
type CodexConfig struct {
	Enabled          bool   `yaml:"enabled" kong:"help='Enable Codex integration',default='false'"`
	Command          string `yaml:"command" kong:"help='Codex command',default='codex'"`
	Model            string `yaml:"model" kong:"help='Codex model',default='gpt-5'"`
	WebSearch        string `yaml:"web_search" kong:"help='Web search mode (disabled/cached/live)',default='disabled'"`
	ReasoningEffort  string `yaml:"reasoning_effort" kong:"help='Reasoning effort (none/minimal/low/medium/high/xhigh)',default='low'"`
	ReasoningSummary string `yaml:"reasoning_summary" kong:"help='Reasoning summary (auto/concise/detailed/none)',default='none'"`
	Verbosity        string `yaml:"verbosity" kong:"help='Model verbosity (low/medium/high)',default='low'"`
	TimeoutSeconds   int    `yaml:"timeout_seconds" kong:"help='Timeout in seconds',default='30'"`
	Sandbox          string `yaml:"sandbox" kong:"help='Sandbox mode (read-only/workspace-write/danger-full-access)',default='read-only'"`
}

// Settings represents the application configuration.
type Settings struct {
	Feeds       []string                 `yaml:"feeds" kong:"help='RSS/Atom Feed URLs',default='https://news.ycombinator.com/rss'"`
	FeedGroups  []subscription.FeedGroup `yaml:"feed_groups"`
	KeyMap      KeyMapConfig             `yaml:"keymap" kong:"embed,prefix='keymap.'"`
	Theme       ThemeConfig              `yaml:"theme" kong:"embed,prefix='theme.'"`
	Codex       CodexConfig              `yaml:"codex" kong:"embed,prefix='codex.'"`
	HistoryFile string                   `yaml:"history_file" kong:"help='History file path'"`
}

// FlattenedFeeds returns grouped feeds first, then ungrouped feeds.
func (s Settings) FlattenedFeeds() []string {
	total := len(s.Feeds)
	for _, group := range s.FeedGroups {
		total += len(group.Feeds)
	}
	result := make([]string, 0, total)
	for _, group := range s.FeedGroups {
		result = append(result, group.Feeds...)
	}
	result = append(result, s.Feeds...)
	return result
}
