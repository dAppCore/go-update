# CLAUDE.md — go-update

This file provides guidance to Claude Code when working with the `go-update` package.

## Package Overview

`go-update` (`forge.lthn.ai/core/go-update`) is a **self-updater library** for Go applications. It supports updates from GitHub releases and generic HTTP endpoints, with configurable startup behaviour and version channel filtering.

## Build & Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a single test
go test -run TestName ./...

# Generate version.go from package.json
go generate ./...

# Vet and lint
go vet ./...
```

## Architecture

### Update Sources

| Source | Description |
|--------|-------------|
| GitHub Releases | Fetches releases via GitHub API, filters by channel (stable/beta/alpha) |
| Generic HTTP | Fetches `latest.json` from a base URL with version + download URL |

### Key Types

- **`UpdateService`** — Configured service that checks for updates on startup
- **`GithubClient`** — Interface for GitHub API interactions (mockable for tests)
- **`Release`** / **`ReleaseAsset`** — GitHub release model
- **`GenericUpdateInfo`** — HTTP update endpoint model

### Testable Function Variables

Core update logic is exposed as `var` function values so tests can replace them:

- `NewGithubClient` — Factory for GitHub client (replace with mock)
- `DoUpdate` — Performs the actual binary update
- `CheckForNewerVersion`, `CheckForUpdates`, `CheckOnly` — GitHub update flow
- `CheckForUpdatesHTTP`, `CheckOnlyHTTP` — HTTP update flow
- `NewAuthenticatedClient` — HTTP client factory (supports `GITHUB_TOKEN`)

### Error Handling

All errors **must** use `coreerr.E()` from `forge.lthn.ai/core/go-log`:

```go
import coreerr "forge.lthn.ai/core/go-log"

return coreerr.E("FunctionName", "what failed", underlyingErr)
```

Never use `fmt.Errorf` or `errors.New`.

### File I/O

Use `forge.lthn.ai/core/go-io` for file operations, not `os.ReadFile`/`os.WriteFile`.

## Coding Standards

- **UK English** in comments and strings
- **Strict types**: All parameters and return types
- **Test naming**: `_Good`, `_Bad`, `_Ugly` suffix pattern
- **License**: EUPL-1.2
