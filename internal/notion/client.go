package notion

import (
	"fmt"

	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/longkey1/gotion/internal/notion/api"
	"github.com/longkey1/gotion/internal/notion/mcp"
	"github.com/longkey1/gotion/internal/notion/types"
)

// Re-export types for convenience
type Client = types.Client
type GetPageOptions = types.GetPageOptions
type SearchOptions = types.SearchOptions
type PageResult = types.PageResult
type SearchResult = types.SearchResult
type PageSummary = types.PageSummary

// NewClient creates a new Notion client based on the config
func NewClient(cfg *config.Config) (Client, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	switch cfg.Backend {
	case config.BackendMCP:
		return mcp.NewClient(cfg.Token)
	case config.BackendAPI, "":
		return api.NewClient(cfg.Token), nil
	default:
		return nil, fmt.Errorf("unknown backend: %s", cfg.Backend)
	}
}
