# Agents Guide

This document provides context for AI agents working on the Reazy codebase.

## Rules
- **Immutable Rules**: The **Rules** section itself `AGENTS.md` must NOT be modified or removed.
- **Documentation**: Always keep `README.md` and `README_ja.md` up to date with the latest features and changes.
- **Testing**: Aim for **100% test coverage**. It MUST exceed **80%** at minimum.
- **Maintenance**: Keep all non-RULE items in this file (`AGENTS.md`) up to date with the latest project state.
- **Review**: When adding code, always perform a self-review and refinement loop to ensure quality and maintainability.

## Project Structure
- `cmd/reazy`: Entry point (`main.go`).
- `internal/domain/reading`: Feed/History domain models.
- `internal/domain/subscription`: Subscription domain model.
- `internal/application/settings`: Application settings types (keymap/theme/etc).
- `internal/application/usecase`: Application services.
- `internal/infrastructure/config`: Configuration storage using `kong` and `yaml.v3`.
- `internal/infrastructure/feed`: RSS parsing logic wrapping `gofeed`.
- `internal/infrastructure/history`: Read-history persistence using JSONL.
- `internal/infrastructure/ai`: AI provider abstraction and concrete clients.
- `internal/presentation/tui`: Bubble Tea Model and View logic.
- `internal/presentation/tui/state`: UI state types.
- `internal/presentation/tui/intent`: Input intent parsing.
- `internal/presentation/tui/update`: State update/reducer logic.
- `internal/presentation/tui/presenter`: View model builders.
- `internal/presentation/tui/components`: Header/sidebar/main/modal UI pieces.
- `internal/presentation/tui/view`: Layout + render orchestration.
- `internal/presentation/tui/view/list`: List item delegates (feed/article).
- `docs/architecture.md`: Current architecture overview.
- `TASKS.md`: Task definitions for `xc`.

## Documentation Notes
- `README.md` と `README_ja.md` は対外向け資料のため、内部設計・実装事情・アーキテクチャ説明は記載しない。

## Coding Standards
- **Architecture**: Follow standard Go project layout.
- **UI**: Use `bubbletea` for TUI. Favor pointer receivers for `Model` to avoid value copying.
- **Testing**: Aim for high coverage without global state dependencies.
- **Error Handling**: Propagate errors or handle gracefully in UI.

## Key Design Decisions
- **Configuration**: Uses `alecthomas/kong` for configuration parsing and defaults, with a custom YAML loader. Settings are loaded via infrastructure store and passed explicitly.
- **Dependency Injection**: Use variables like `feed.ParserFunc` to mock external dependencies (network calls) in tests.
- **AI Insights**: Insight generation belongs to Application usecases and depends on abstract text-generation clients. Infrastructure only provides concrete AI clients (currently Codex CLI via `codex.*` config).
- **News Tab**: `internal://news` is a built-in virtual feed that shows AI-generated daily digest topic cards. Digest items are stored as `news_digest` and kept as date-grouped history.
- **Date Sections**: Date section headers are applied to normal article lists (`All Feeds` / `Bookmarks` / each feed), not to `News`.

## Tools
- `xc`: Task runner. Use `xc [task]` to run predefined tasks.
