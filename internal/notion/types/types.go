package types

import "context"

// Client defines the interface for Notion API operations
type Client interface {
	// GetPage retrieves a page by ID
	GetPage(ctx context.Context, pageID string, opts *GetPageOptions) (*PageResult, error)

	// Search searches for pages
	Search(ctx context.Context, query string, opts *SearchOptions) (*SearchResult, error)
}

// GetPageOptions contains options for GetPage
type GetPageOptions struct {
	FilterProperties []string
}

// SearchOptions contains options for Search
type SearchOptions struct {
	PageSize    int
	StartCursor string
	Sort        string // "ascending" or "descending"
}

// PageResult represents the result of GetPage
type PageResult struct {
	// For API client
	ID         string
	Title      string
	URL        string
	Properties map[string]string

	// For MCP client (raw content)
	Content string

	// Source indicates which client produced this result
	Source string // "api" or "mcp"
}

// SearchResult represents the result of Search
type SearchResult struct {
	// For API client
	Pages      []PageSummary
	HasMore    bool
	NextCursor string

	// For MCP client (raw content)
	Content string

	// Source indicates which client produced this result
	Source string // "api" or "mcp"
}

// PageSummary represents a summary of a page in search results
type PageSummary struct {
	ID    string
	Title string
	URL   string
}
