# gotion

A CLI tool for interacting with the Notion API.

## Installation

```bash
go install github.com/longkey1/gotion@latest
```

Or clone and build:

```bash
git clone https://github.com/longkey1/gotion.git
cd gotion
go build -o gotion .
```

## Configuration

gotion supports two backends: **MCP** and **API**. Configure via environment variables or config file.

### Backend Selection

Set `backend` to choose which Notion API to use:

```bash
# Environment variable
export GOTION_BACKEND="mcp"  # or "api"

# Or config file (~/.config/gotion/config.toml)
backend = "mcp"
```

| Backend | Description |
|---------|-------------|
| `mcp` | MCP API with Dynamic Client Registration (no setup required) |
| `api` | Traditional REST API (requires client_id and client_secret) |

## Authentication

### MCP Backend (Recommended)

No pre-configuration required. Uses Dynamic Client Registration (RFC 7591).

```bash
# Set backend to mcp
export GOTION_BACKEND="mcp"

# Run authentication
gotion auth
```

A browser window will open for Notion authorization. Credentials are saved to `~/.config/gotion/token.json`.

### API Backend

Requires creating a Notion Integration.

1. Create a Public Integration at [Notion Integrations](https://www.notion.so/my-integrations)
2. Set Redirect URI to `http://localhost:8080/callback`
3. Get Client ID and Client Secret

Configure credentials:

```bash
# Environment variables
export GOTION_BACKEND="api"
export GOTION_CLIENT_ID="your-client-id"
export GOTION_CLIENT_SECRET="your-client-secret"

# Or config file (~/.config/gotion/config.toml)
backend = "api"
client_id = "your-client-id"
client_secret = "your-client-secret"
```

Run authentication:

```bash
gotion auth
```

### Direct Token

Use an Internal Integration token directly (skips OAuth):

```bash
export GOTION_TOKEN="secret_xxxxxxxx"
# or
export NOTION_TOKEN="secret_xxxxxxxx"
```

## Usage

### Search Pages

```bash
# Search with keyword
gotion list -q "search keyword"

# Limit results
gotion list -q "search keyword" -n 20

# Output as JSON (API backend only)
gotion list -q "search keyword" -f json
```

### Get Page

```bash
# Get page by ID or URL
gotion get <page_id>

# Output as JSON (API backend only)
gotion get <page_id> -f json

# Filter specific properties
gotion get <page_id> --filter-properties "title,status"
```

## Output Formats

| Backend | Default | JSON |
|---------|---------|------|
| MCP | Markdown | Not supported |
| API | Markdown | `--format json` |

Both backends output Markdown by default. The `--format json` option is only available with the API backend.

## Commands

| Command | Description |
|---------|-------------|
| `auth` | Authenticate with Notion |
| `config` | Show current configuration |
| `list` | Search and list pages |
| `get` | Get page details |
| `version` | Show version info |

## Environment Variables

All config file settings can be overridden with environment variables:

| Environment Variable | Config Key | Description |
|---------------------|------------|-------------|
| `GOTION_BACKEND` | `backend` | API backend (`api` or `mcp`) |
| `GOTION_CLIENT_ID` | `client_id` | OAuth client ID |
| `GOTION_CLIENT_SECRET` | `client_secret` | OAuth client secret |
| `GOTION_TOKEN` | - | Direct API token |
| `NOTION_TOKEN` | - | Direct API token (fallback) |

Priority: Environment variables > Config file > Token file

## Files

| File | Description |
|------|-------------|
| `~/.config/gotion/config.toml` | Configuration settings |
| `~/.config/gotion/token.json` | OAuth tokens |

## License

MIT
