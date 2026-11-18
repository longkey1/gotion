package gotion

import (
	"fmt"
	"io"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// PageOutput is the intermediate structure for page formatting
type PageOutput struct {
	Title      string
	URL        string
	Properties map[string]any // emitted after title and url, keys sorted
	Content    string
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
	var sb strings.Builder

	sb.WriteString("---\n")
	if err := writeYAML(&sb, pageFrontmatter{Title: output.Title, URL: output.URL}); err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}
	if len(output.Properties) > 0 {
		if err := writeYAML(&sb, output.Properties); err != nil {
			return "", fmt.Errorf("failed to marshal frontmatter properties: %w", err)
		}
	}
	sb.WriteString("---\n\n")

	if output.Content != "" {
		sb.WriteString(output.Content)
	}

	return sb.String(), nil
}

func writeYAML(w io.Writer, v any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return enc.Close()
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
