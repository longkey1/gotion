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
	fmt.Fprintf(&sb, "title: %q\n", output.Title)
	fmt.Fprintf(&sb, "url: %s\n", output.URL)
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
		fmt.Fprintf(&sb, "- [%s](%s)\n", page.Title, page.URL)
	}

	if output.HasMore && output.NextCursor != "" {
		fmt.Fprintf(&sb, "\n_More results available (cursor: %s)_\n", output.NextCursor)
	}

	return sb.String()
}
