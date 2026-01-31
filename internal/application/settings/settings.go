// Package settings defines application-level configuration data.
package settings

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

// Settings represents the application configuration.
type Settings struct {
	Feeds       []string     `yaml:"feeds" kong:"help='RSS Feed URLs',default='https://news.ycombinator.com/rss'"`
	KeyMap      KeyMapConfig `yaml:"keymap" kong:"embed,prefix='keymap.'"`
	Theme       ThemeConfig  `yaml:"theme" kong:"embed,prefix='theme.'"`
	HistoryFile string       `yaml:"history_file" kong:"help='History file path'"`
}
