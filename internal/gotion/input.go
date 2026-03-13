package gotion

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ParsedInput represents parsed input from stdin or a file
type ParsedInput struct {
	Properties map[string]interface{}
	Content    string
}

// ParseInput parses input from a reader.
// It auto-detects the format:
//   - JSON: if input starts with '{'
//   - Markdown with frontmatter: if input starts with '---'
//   - Plain Markdown: otherwise (no properties)
func ParseInput(r io.Reader) (*ParsedInput, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil, fmt.Errorf("empty input")
	}

	if strings.HasPrefix(text, "{") {
		return parseJSONInput(text)
	}

	if strings.HasPrefix(text, "---") {
		return parseFrontmatterInput(text)
	}

	// Plain markdown content, no properties
	return &ParsedInput{
		Content: text,
	}, nil
}

func parseJSONInput(text string) (*ParsedInput, error) {
	var raw struct {
		Properties map[string]interface{} `json:"properties"`
		Content    string                 `json:"content"`
	}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON input: %w", err)
	}

	return &ParsedInput{
		Properties: raw.Properties,
		Content:    raw.Content,
	}, nil
}

func parseFrontmatterInput(text string) (*ParsedInput, error) {
	// Split on "---" delimiter
	// Expected format:
	// ---
	// key: value
	// ---
	// content...
	lines := strings.SplitN(text, "\n", 2)
	if len(lines) < 2 {
		return &ParsedInput{Content: text}, nil
	}

	// Find the closing "---"
	rest := lines[1]
	closingIdx := strings.Index(rest, "\n---")
	if closingIdx == -1 {
		// No closing delimiter, treat entire thing as content
		return &ParsedInput{Content: text}, nil
	}

	frontmatter := rest[:closingIdx]
	content := strings.TrimSpace(rest[closingIdx+4:]) // skip "\n---"

	props := parseFrontmatterProperties(frontmatter)

	// Extract title from frontmatter into properties
	result := &ParsedInput{
		Properties: make(map[string]interface{}),
		Content:    content,
	}

	for k, v := range props {
		if k == "title" {
			result.Properties["title"] = v
		} else if k != "url" {
			// Skip url (it's metadata, not a property to set)
			result.Properties[k] = v
		}
	}

	if len(result.Properties) == 0 {
		result.Properties = nil
	}

	return result, nil
}

// parseFrontmatterProperties parses simple "key: value" lines from frontmatter
func parseFrontmatterProperties(frontmatter string) map[string]string {
	props := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(frontmatter))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove surrounding quotes
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		props[key] = value
	}

	return props
}
