package gotion

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// ParsedInput represents parsed input from stdin or a file
type ParsedInput struct {
	Properties map[string]any
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
		Properties map[string]any `json:"properties"`
		Content    string         `json:"content"`
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

	props, err := parseFrontmatterProperties(frontmatter)
	if err != nil {
		return nil, err
	}

	// url is page metadata emitted by "get --format markdown", not a settable property
	delete(props, "url")

	result := &ParsedInput{
		Properties: props,
		Content:    content,
	}

	if len(result.Properties) == 0 {
		result.Properties = nil
	}

	return result, nil
}

// parseFrontmatterProperties parses the frontmatter block as a YAML mapping
func parseFrontmatterProperties(frontmatter string) (map[string]any, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &doc); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Empty frontmatter yields a zero node
	if len(doc.Content) == 0 {
		return map[string]any{}, nil
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("failed to parse frontmatter: expected key-value mapping")
	}

	value, err := yamlNodeValue(root)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	props, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to parse frontmatter: expected key-value mapping")
	}

	return props, nil
}

// yamlNodeValue converts a YAML node to a Go value, keeping timestamps as
// their original text: decoding them to time.Time would re-serialize dates
// like 2024-01-02 as RFC 3339 datetimes when sent to Notion.
func yamlNodeValue(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.AliasNode:
		return yamlNodeValue(node.Alias)
	case yaml.MappingNode:
		m := make(map[string]any, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			var key string
			if err := node.Content[i].Decode(&key); err != nil {
				return nil, err
			}
			v, err := yamlNodeValue(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			m[key] = v
		}
		return m, nil
	case yaml.SequenceNode:
		s := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			v, err := yamlNodeValue(child)
			if err != nil {
				return nil, err
			}
			s = append(s, v)
		}
		return s, nil
	case yaml.ScalarNode:
		if node.ShortTag() == "!!timestamp" {
			return node.Value, nil
		}
	}

	var v any
	if err := node.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
