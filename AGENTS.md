# AGENTS.md

This file provides guidance to AI coding agents (Claude Code, etc.) when working with code in this repository.

## Project Overview

`gotion` is a CLI for the Notion API built with Cobra: it searches, gets, creates, and updates Notion pages through one of two interchangeable backends — the official REST API (`api`) or the Notion MCP server (`mcp`) — selected via the `backend` config value. Authentication is OAuth (browser flow with a local callback server) or a direct integration token; tokens are stored in `~/.config/gotion/token.json` and MCP tokens are auto-refreshed before API commands run.

## Build Commands

```sh
make build   # Build binary to ./bin/gotion
make test    # Run tests
make fmt     # Format code
make vet     # Vet code
make tidy    # Tidy dependencies
make clean   # Remove build artifacts
```

Binary name is read from `.product_name`.

## Release

```sh
make release type=patch|minor|major            # dry run (default)
make release type=patch dryrun=false           # create and push tag
make re-release [tag=vX.Y.Z] dryrun=false      # re-release an existing tag
```

Pushing a `v*` tag triggers `.github/workflows/gorelease.yml`, which builds release binaries with GoReleaser and uploads them to GitHub Releases.

## Architecture

- `main.go` — entry point, calls `cmd.Execute()`
- `cmd/` — Cobra commands
  - `root.go` — root command. `PersistentPreRunE` refreshes an expired MCP token before any command except `auth`/`config`/`version`/`help`/`completion` (`skipTokenRefresh` walks the command's parent chain); refresh tolerates another process having already refreshed the token file
  - `get.go` — `gotion get <page_id>` accepts an ID or Notion URL (`gotion.ExtractPageID`), `--filter-properties` (comma-separated, API backend), `--format json|markdown` (markdown = YAML frontmatter with title/url + content)
  - `list.go` — `gotion list` searches pages: `-q` query, `-n` page size (clamped to 1..100, default 10), `--sort ascending|descending`, `--cursor` for pagination; prints the backend's raw JSON
  - `create.go` — `gotion create` builds a page from stdin or `--file` via `gotion.ParseInput`; `--title` overrides the input's title property, `--parent` + `--parent-type page_id|database_id|data_source_id` set the parent (parent ID also goes through `ExtractPageID`). MCP backend only — the API client returns an "unsupported" error
  - `update.go` — `gotion update <page_id>` updates properties and/or content from the same input formats; `--properties-only` / `--content-only` are mutually exclusive. MCP backend only
  - `auth.go` — `gotion auth` runs the OAuth flow chosen by `backend`: MCP uses discovery (RFC 9728/8414) + Dynamic Client Registration (RFC 7591) + PKCE (RFC 7636) with a fixed callback on port 9998; API uses classic OAuth with client_id/client_secret and Basic auth on the token endpoint (`-p/--port`, default 8080). Both open the browser, wait on a local callback server (5-minute timeout, CSRF state check), and save the token via `config.SaveToken`
  - `config.go` — `gotion config` prints the effective configuration with masked secrets (`maskToken`) and which sources (env vars, config file, token file) are present
  - `version.go` — prints `version.Info()`
- `internal/gotion/` — CLI-side core logic (backend-independent)
  - `page.go` — `ExtractPageID` pulls a 32-hex or hyphenated UUID out of notion.so/notion.site URLs and strips hyphens; non-URL input just has hyphens stripped
  - `input.go` — `ParseInput` auto-detects create/update input: leading `{` = JSON (`properties` + `content`), leading `---` = Markdown with simple `key: value` frontmatter (quotes stripped; `title` becomes a property, `url` is dropped as metadata; an unterminated frontmatter block falls back to plain content), otherwise plain Markdown content with no properties
  - `format.go` — `FormatPage` (YAML-frontmatter Markdown) and `FormatSearch` (Markdown link list, optional next-cursor footer) render the intermediate `PageOutput`/`SearchOutput` structures
  - `auth.go` — `CallbackServer`, a localhost HTTP server for the OAuth redirect: validates state, captures the authorization code, handles the `error` query parameter, and unblocks `Start` once the callback arrives (or the context is cancelled)
- `internal/gotion/config/` — configuration and token storage. `Load` uses Viper with priority env vars (`GOTION_BACKEND`, `GOTION_API_TOKEN`, `GOTION_API_CLIENT_ID`, `GOTION_API_CLIENT_SECRET`) > `~/.config/gotion/config.toml` > `NOTION_TOKEN` env fallback > token file. `TokenData` (JSON, written 0600) tracks backend, access/refresh tokens, and expiry; `IsTokenExpired` uses a 5-minute margin and `NeedsRefresh` additionally requires a refresh token. `Validate` requires a token; `ValidateOAuth` requires client_id/client_secret
- `internal/notion/` — `NewClient(cfg)` picks the backend implementation: `mcp` → `mcp.Client`, `api` or empty → `api.Client`, anything else is an error. Re-exports the `types` names for convenience
- `internal/notion/types/` — the `Client` interface (GetPage/Search/CreatePage/UpdatePage/FormatPage/FormatSearch) and shared option/result structs both backends implement
- `internal/notion/api/` — REST backend against `api.notion.com/v1` (Notion-Version 2022-06-28). `GetPage` fetches page metadata plus all block children (paginated, recursing into blocks with `has_children`) and combines them into one JSON document; `extractTitle`/`extractProperties` flatten title and rich_text properties. `Search` posts to `/search` filtered to pages. Create/Update return "not supported" errors. `oauth.go` implements the classic Notion OAuth code exchange
- `internal/notion/mcp/` — MCP backend against `mcp.notion.com/mcp` speaking JSON-RPC 2.0 over HTTP with SSE response support (`parseSSEResponse`) and `Mcp-Session-Id` session tracking. Operations map to MCP tools: `notion-fetch`, `notion-search`, `notion-create-pages`, `notion-update-page` (properties and content are two separate calls: `update_properties` then `replace_content`). `oauth.go` implements discovery, Dynamic Client Registration, PKCE, code exchange, and `RefreshToken`
- `internal/version/` — version info injected via ldflags at build time

## Testing

Tests use only the standard library (table-driven with `t.Run` subtests, `t.TempDir()` for filesystem state, `httptest.Server` for HTTP; no live network calls):

- `internal/gotion/page_test.go` — `ExtractPageID` URL/ID parsing
- `internal/gotion/format_test.go` — `FormatPage` / `FormatSearch` rendering
- `internal/gotion/input_test.go` — `ParseInput` format detection and frontmatter parsing
- `internal/gotion/auth_test.go` — `CallbackServer` success, error, state-mismatch, and cancellation paths (loopback HTTP)
- `internal/gotion/config/config_test.go` — config precedence (env > file > NOTION_TOKEN > token file, via `t.Setenv("HOME", ...)` redirection), token expiry logic, token save/load/delete round-trip and 0600 permissions, validation
- `internal/notion/client_test.go` — backend selection in `NewClient`
- `internal/notion/api/client_test.go` — property/title extraction, `doRequest` success and error mapping (httptest), output conversion, unsupported create/update
- `internal/notion/api/oauth_test.go` — authorization URL construction
- `internal/notion/mcp/client_test.go` — JSON-RPC error parsing, SSE response parsing, page metadata extraction
- `internal/notion/mcp/oauth_test.go` — PKCE generation (RFC 7636 relationship), endpoint discovery, dynamic client registration, and code exchange against an httptest metadata server
- `cmd/root_test.go` — `skipTokenRefresh` command classification
- `cmd/config_test.go` — `maskToken`

## Key behavior

- Backend selection: `backend = "mcp"` or `"api"` (empty defaults to `api`). Create and update only work on the MCP backend; the REST backend is read-only (get/list)
- Config precedence: environment variables > `~/.config/gotion/config.toml` > `NOTION_TOKEN` > saved token file. A direct integration token (`GOTION_API_TOKEN`/`NOTION_TOKEN`) skips OAuth entirely
- `get`, `create`, and `update` accept Notion URLs anywhere a page ID is expected; IDs are normalized by stripping hyphens
- `create`/`update` input format is auto-detected (JSON / frontmatter Markdown / plain Markdown); frontmatter parsing is intentionally simple line-based `key: value`, not full YAML
- MCP tokens are refreshed automatically in `PersistentPreRunE` when expired (5-minute margin) and a refresh token exists; API-backend tokens are never auto-refreshed
- `list`/`get` print the backend's raw JSON by default, so output shape differs between backends; `get --format markdown` produces a stable frontmatter+content form suitable for a get → edit → update round-trip
