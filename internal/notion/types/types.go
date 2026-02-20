package types

import "context"

// Client defines the interface for Notion API operations
type Client interface {
	// GetPage retrieves a page by ID
	GetPage(ctx context.Context, pageID string, opts *GetPageOptions) (*PageResult, error)

	// Search searches for pages
	Search(ctx context.Context, query string, opts *SearchOptions) (*SearchResult, error)

	// FormatPage formats a page result as JSON string
	FormatPage(result *PageResult) (string, error)

	// FormatSearch formats a search result as JSON string
	FormatSearch(result *SearchResult) (string, error)
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
	ID      string
	Title   string
	URL     string
	Content string            // Markdown content
	RawJSON []byte            // Raw JSON (API only)
	Props   map[string]string // Properties
	Source  string            // "api" or "mcp"
}

// SearchResult represents the result of Search
type SearchResult struct {
	Pages      []PageSummary
	HasMore    bool
	NextCursor string
	Content    string // Markdown content
	RawJSON    []byte // Raw JSON (API only)
	Source     string // "api" or "mcp"
}

// PageSummary represents a summary of a page in search results
type PageSummary struct {
	ID    string
	Title string
	URL   string
}
