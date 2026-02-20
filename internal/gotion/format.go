package gotion

import (
	"fmt"
	"strings"
)

// PageOutput is the intermediate structure for page formatting
type PageOutput struct {
	Title   string
	URL     string
	Content string
}

// SearchPageItem represents a single page in search results
type SearchPageItem struct {
	Title string
	URL   string
}

// SearchOutput is the intermediate structure for search result formatting
type SearchOutput struct {
	Pages      []SearchPageItem
	HasMore    bool
	NextCursor string
}

// FormatPage formats a PageOutput as Markdown with YAML frontmatter
func FormatPage(output *PageOutput) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", output.Title))
	sb.WriteString(fmt.Sprintf("url: %s\n", output.URL))
	sb.WriteString("---\n\n")

	if output.Content != "" {
		sb.WriteString(output.Content)
	}

	return sb.String()
}

// FormatSearch formats a SearchOutput as Markdown
func FormatSearch(output *SearchOutput) string {
	var sb strings.Builder

	for _, page := range output.Pages {
		sb.WriteString(fmt.Sprintf("- [%s](%s)\n", page.Title, page.URL))
	}

	if output.HasMore && output.NextCursor != "" {
		sb.WriteString(fmt.Sprintf("\n_More results available (cursor: %s)_\n", output.NextCursor))
	}

	return sb.String()
}
