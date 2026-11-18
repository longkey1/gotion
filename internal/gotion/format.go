package gotion

import (
	"fmt"
	"strings"

	yaml "go.yaml.in/yaml/v3"
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

type pageFrontmatter struct {
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
}

// FormatPage formats a PageOutput as Markdown with YAML frontmatter
func FormatPage(output *PageOutput) (string, error) {
	frontmatter, err := yaml.Marshal(pageFrontmatter{
		Title: output.Title,
		URL:   output.URL,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.Write(frontmatter)
	sb.WriteString("---\n\n")

	if output.Content != "" {
		sb.WriteString(output.Content)
	}

	return sb.String(), nil
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
