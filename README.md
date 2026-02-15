# Reazy

Reazy (Read + Lazy) is a modern, terminal-based RSS/Atom reader built with Go and Bubble Tea. It provides a simple, Vim-like interface for managing and reading your favorite feeds directly from the command line.

## Features

- **TUI Interface**: Clean and responsive terminal UI.
- **Feed Management**: Add and delete RSS/Atom feeds easily.
- **Reading**: Browse feed items and open full articles in your default browser.
- **Vim Bindings**: Navigation with `j`, `k`, `h`, `l`.
- **Customizable**: Configurable keybindings and feed list via YAML.
- **Updates**: Pull-to-refresh support.
- **Read Status**: Tracks read articles and dims them.
- **All Feeds**: View articles from all feeds in a unified timeline.
- **Date Sections in Lists**: `All Feeds` / `Bookmarks` / each feed view are grouped by date.
- **News Tab (AI Digest)**: Build daily AI digest topics from today's articles and keep digest history grouped by date. Refreshing News appends new topics without deleting older ones from the same day.
- **SQLite History Store**: Read state/bookmarks/AI metadata are persisted in SQLite for faster startup and updates.
- **AI Summary View**: In the detail screen, AI summary and article body are clearly separated for easier reading.
- **AI Insights (Optional)**: Generate article summaries and tags via Codex CLI.
- **Status Footer**: AI generation status and feed timeout/failure notices are shown in the footer.

## Installation

Requires Go 1.26 or later when building from source.

### From Source
```bash
go install github.com/tesso57/reazy/cmd/reazy@latest
```

Or clone and build:
```bash
git clone https://github.com/tesso57/reazy.git
cd reazy
go build ./cmd/reazy
```

## Usage

Run the application:
```bash
reazy
```

In the feed sidebar, select `* News` to open AI digest history grouped by date.  
Today's digest is generated from your registered feeds and cached for the day.  
Manual refresh in `News` regenerates today's digest and keeps previous topics for that date.  
In normal feed views (`All Feeds` / `Bookmarks` / each feed), articles are grouped by date sections.
If some feeds are slow, Reazy shows available results first and reports timeout count in the footer.

### Keybindings (Default)
- **Navigation**:
  - `k` / `↑`: Up
  - `j` / `↓`: Down
  - `h` / `←`: Back / Focus Feeds
  - `l` / `→` / `Enter`: Open selected item (article, digest topic, or link in detail)
- **Actions**:
  - `a`: Add Feed
  - `x`: Delete Feed
  - `r`: Refresh current feed (`News` regenerates today's digest and keeps previous topics for the date)
  - `b`: Toggle Bookmark
  - `s`: Generate AI Summary/Tags (article/detail)
  - `S`: Toggle AI Summary visibility (detail view)
  - `?`: Toggle Help
  - `q`: Quit

## Configuration
Configuration is stored in `$XDG_CONFIG_HOME/reazy/config.yaml` (usually `~/.config/reazy/config.yaml`).
`history_file` defaults to `~/.local/share/reazy/history.db`.
If you still have a `.jsonl` path, Reazy automatically uses `history.db` in the same directory.

Example:
```yaml
feeds:
  - https://news.ycombinator.com/rss
  - https://github.com/golang/go/releases.atom
keymap:
  up: k
  down: j
  ...
history_file: /Users/you/.local/share/reazy/history.db
codex:
  enabled: false
  command: codex
  model: gpt-5
  web_search: disabled
  reasoning_effort: low
  reasoning_summary: none
  verbosity: low
  timeout_seconds: 30
  sandbox: read-only
```

### Codex Integration (Optional)
If Codex CLI is installed and authenticated, you can enable AI insights:

```yaml
codex:
  enabled: true
```

Then select an article and press `s` in article/detail view to generate:
- a Japanese summary readable in about 3 minutes
- English topic tags

You can also open `* News` to view date-grouped AI digest history (including past days).

## Alternatives
There are other RSS readers available:
- [eilmeldung](https://github.com/christo-auer/eilmeldung)
- [russ](https://github.com/ckampfe/russ)

## Tasks

### build

Build the application binary.

```bash
go build -o reazy ./cmd/reazy
```

### run

Run the Reazy application.

```bash
go run ./cmd/reazy
```

### test

Run all unit tests.

```bash
go test ./...
```

### lint

Run static analysis using golangci-lint.

```bash
golangci-lint run ./...
```

### cover

Run tests with coverage and open HTML report.

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
rm coverage.out
```

### clean

Remove coverage artifacts.

```bash
go clean
rm -f coverage.out
```
