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
export GOTION_API_CLIENT_ID="your-client-id"
export GOTION_API_CLIENT_SECRET="your-client-secret"

# Or config file (~/.config/gotion/config.toml)
backend = "api"
api_client_id = "your-client-id"
api_client_secret = "your-client-secret"
```

Run authentication:

```bash
gotion auth
```

### Direct Token

Use an Internal Integration token directly (skips OAuth):

```bash
export GOTION_API_TOKEN="secret_xxxxxxxx"
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
```

### Get Page

```bash
# Get page by ID or URL
gotion get <page_id>

# Output as Markdown (MCP backend: frontmatter + content)
gotion get <page_id> --format markdown

# Output as JSON (default)
gotion get <page_id> --format json

# Filter specific properties
gotion get <page_id> --filter-properties "title,status"
```

### Create Page

Requires MCP backend.

```bash
# Create from Markdown with frontmatter via stdin
echo '---
title: "New Page"
---

# Hello
Page content here' | gotion create --parent <parent_id>

# Create from file
gotion create --parent <parent_id> --file page.md

# Create with title flag
gotion create --parent <parent_id> --title "New Page" --file content.md

# Create from JSON
echo '{"properties":{"title":"New Page"},"content":"# Hello"}' | gotion create --parent <parent_id>

# Specify parent type (default: page_id)
gotion create --parent <database_id> --parent-type database_id --file page.md
```

### Update Page

Requires MCP backend.

```bash
# Update from file (properties + content)
gotion update <page_id> --file page.md

# Update from stdin
cat page.md | gotion update <page_id>

# Update content only
gotion update <page_id> --content-only --file page.md

# Update properties only
gotion update <page_id> --properties-only --file page.md
```

### Get → Edit → Update Workflow

```bash
# Export page as Markdown
gotion get <page_id> --format markdown > page.md

# Edit in your favorite editor
vim page.md

# Push changes back
gotion update <page_id> --file page.md
```

## Input Formats

`create` and `update` commands accept input via stdin or `--file`. The format is auto-detected:

### Markdown with YAML Frontmatter

```markdown
---
title: "Page Title"
Status: "In Progress"
---

# Heading
Body text
```

### JSON

```json
{
  "properties": {"title": "Page Title"},
  "content": "# Heading\nBody text"
}
```

### Plain Markdown

If input has no frontmatter and is not JSON, it is treated as content only (no properties).

## Output Formats

The `get` command supports `--format` flag:

| Format | Description |
|--------|-------------|
| `json` (default) | Raw JSON response |
| `markdown` | Markdown with YAML frontmatter (title, url) |

## Commands

| Command | Description |
|---------|-------------|
| `auth` | Authenticate with Notion |
| `config` | Show current configuration |
| `list` | Search and list pages |
| `get` | Get page details |
| `create` | Create a new page (MCP only) |
| `update` | Update an existing page (MCP only) |
| `version` | Show version info |

## Environment Variables

All config file settings can be overridden with environment variables:

| Environment Variable | Config Key | Description |
|---------------------|------------|-------------|
| `GOTION_BACKEND` | `backend` | API backend (`api` or `mcp`) |
| `GOTION_API_CLIENT_ID` | `api_client_id` | OAuth client ID |
| `GOTION_API_CLIENT_SECRET` | `api_client_secret` | OAuth client secret |
| `GOTION_API_TOKEN` | `api_token` | Direct API token |
| `NOTION_TOKEN` | - | Direct API token (fallback) |

Priority: Environment variables > Config file > Token file

## Files

| File | Description |
|------|-------------|
| `~/.config/gotion/config.toml` | Configuration settings |
| `~/.config/gotion/token.json` | OAuth tokens |

## License

MIT
