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
- `internal/config`: Configuration management using `kong` and `yaml.v3`.
- `internal/feed`: RSS parsing logic wrapping `gofeed`.
- `internal/ui`: Bubble Tea Model and View logic.
- `TASKS.md`: Task definitions for `xc`.

## Coding Standards
- **Architecture**: Follow standard Go project layout.
- **UI**: Use `bubbletea` for TUI. Favor pointer receivers for `Model` to avoid value copying.
- **Testing**: Aim for high coverage without global state dependencies.
- **Error Handling**: Propagate errors or handle gracefully in UI.

## Key Design Decisions
- **Configuration**: Uses `alecthomas/kong` for configuration parsing and defaults, with a custom YAML loader. Global state is avoided; `Config` structs are passed explicitly.
- **Dependency Injection**: Use variables like `feed.ParserFunc` to mock external dependencies (network calls) in tests.

## Tools
- `xc`: Task runner. Use `xc [task]` to run predefined tasks.
