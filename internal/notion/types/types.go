package types

import "context"

// Client defines the interface for Notion API operations
type Client interface {
	// GetPage retrieves a page by ID
	GetPage(ctx context.Context, pageID string, opts *GetPageOptions) (*PageResult, error)

	// Search searches for pages
	Search(ctx context.Context, query string, opts *SearchOptions) (*SearchResult, error)

	// CreatePage creates a new page
	CreatePage(ctx context.Context, opts *CreatePageOptions) (*CreatePageResult, error)

	// UpdatePage updates an existing page
	UpdatePage(ctx context.Context, pageID string, opts *UpdatePageOptions) (*UpdatePageResult, error)

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

// Parent represents the parent of a page
type Parent struct {
	Type string // "page_id", "database_id", "data_source_id"
	ID   string
}

// CreatePageOptions contains options for CreatePage
type CreatePageOptions struct {
	Parent     *Parent
	Properties map[string]interface{}
	Content    string
}

// UpdatePageOptions contains options for UpdatePage
type UpdatePageOptions struct {
	Properties map[string]interface{}
	Content    *string
}

// CreatePageResult represents the result of CreatePage
type CreatePageResult struct {
	RawJSON []byte
	Source  string
}

// UpdatePageResult represents the result of UpdatePage
type UpdatePageResult struct {
	RawJSON []byte
	Source  string
}
