# Reazy

Reazy (Read + Lazy) is a modern, terminal-based RSS reader built with Go and Bubble Tea. It provides a simple, Vim-like interface for managing and reading your favorite RSS feeds directly from the command line.

## Features

- **TUI Interface**: Clean and responsive terminal UI.
- **Feed Management**: Add and delete RSS feeds easily.
- **Reading**: Browse feed items and open full articles in your default browser.
- **Vim Bindings**: Navigation with `j`, `k`, `h`, `l`.
- **Customizable**: Configurable keybindings and feed list via YAML.
- **Updates**: Pull-to-refresh support.
- **Read Status**: Tracks read articles and dims them.
- **All Feeds**: View articles from all feeds in a unified timeline.
- **Article Preview**: View article summaries directly in the terminal to save time.

## Installation

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

### Keybindings (Default)
- **Navigation**:
  - `k` / `↑`: Up
  - `j` / `↓`: Down
  - `h` / `←`: Back / Focus Feeds
  - `l` / `→` / `Enter`: View Summary / Open Link
- **Actions**:
  - `a`: Add Feed
  - `x`: Delete Feed
  - `r`: Refresh Feed
  - `?`: Toggle Help
  - `q`: Quit

## Configuration
Configuration is stored in `$XDG_CONFIG_HOME/reazy/config.yaml` (usually `~/.config/reazy/config.yaml`).

Example:
```yaml
feeds:
  - https://news.ycombinator.com/rss
keymap:
  up: k
  down: j
  ...
```

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
