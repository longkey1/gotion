package gotion

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    *ParsedInput
		wantErr bool
	}{
		{
			name:  "JSON with properties and content",
			input: `{"properties": {"title": "My Page", "status": "done"}, "content": "# Hello"}`,
			want: &ParsedInput{
				Properties: map[string]any{"title": "My Page", "status": "done"},
				Content:    "# Hello",
			},
		},
		{
			name:  "JSON with content only",
			input: `{"content": "just content"}`,
			want:  &ParsedInput{Content: "just content"},
		},
		{
			name:    "invalid JSON",
			input:   `{"properties": `,
			wantErr: true,
		},
		{
			name:  "frontmatter with title and properties",
			input: "---\ntitle: My Page\nstatus: done\n---\n\n# Body\n\nText.",
			want: &ParsedInput{
				Properties: map[string]any{"title": "My Page", "status": "done"},
				Content:    "# Body\n\nText.",
			},
		},
		{
			name:  "frontmatter url key is skipped",
			input: "---\ntitle: My Page\nurl: https://www.notion.so/abc\n---\ncontent",
			want: &ParsedInput{
				Properties: map[string]any{"title": "My Page"},
				Content:    "content",
			},
		},
		{
			name:  "frontmatter with double-quoted value",
			input: "---\ntitle: \"Quoted Title\"\n---\nbody",
			want: &ParsedInput{
				Properties: map[string]any{"title": "Quoted Title"},
				Content:    "body",
			},
		},
		{
			name:  "frontmatter with single-quoted value",
			input: "---\ntitle: 'Quoted Title'\n---\nbody",
			want: &ParsedInput{
				Properties: map[string]any{"title": "Quoted Title"},
				Content:    "body",
			},
		},
		{
			name:  "frontmatter value containing colon",
			input: "---\nurl2: https://example.com\n---\nbody",
			want: &ParsedInput{
				Properties: map[string]any{"url2": "https://example.com"},
				Content:    "body",
			},
		},
		{
			name:  "frontmatter with only url yields nil properties",
			input: "---\nurl: https://www.notion.so/abc\n---\nbody",
			want:  &ParsedInput{Content: "body"},
		},
		{
			name:  "frontmatter without closing delimiter is plain content",
			input: "--- this is not frontmatter",
			want:  &ParsedInput{Content: "--- this is not frontmatter"},
		},
		{
			name:  "unterminated frontmatter is plain content",
			input: "---\ntitle: My Page\nno closing delimiter",
			want:  &ParsedInput{Content: "---\ntitle: My Page\nno closing delimiter"},
		},
		{
			name:  "plain markdown",
			input: "# Just Markdown\n\nNo properties here.",
			want:  &ParsedInput{Content: "# Just Markdown\n\nNo properties here."},
		},
		{
			name:  "leading whitespace is trimmed before detection",
			input: "\n\n  {\"content\": \"x\"}\n",
			want:  &ParsedInput{Content: "x"},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace-only input",
			input:   "  \n\t\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseInput(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseInput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseInput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseFrontmatterProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		frontmatter string
		want        map[string]string
	}{
		{
			name:        "simple key values",
			frontmatter: "title: Hello\nstatus: done",
			want:        map[string]string{"title": "Hello", "status": "done"},
		},
		{
			name:        "blank lines and lines without colon are skipped",
			frontmatter: "title: Hello\n\nnot-a-pair\nstatus: done",
			want:        map[string]string{"title": "Hello", "status": "done"},
		},
		{
			name:        "surrounding whitespace is trimmed",
			frontmatter: "  title  :   spaced out  ",
			want:        map[string]string{"title": "spaced out"},
		},
		{
			name:        "quotes are stripped",
			frontmatter: "a: \"double\"\nb: 'single'",
			want:        map[string]string{"a": "double", "b": "single"},
		},
		{
			name:        "single character value keeps quote-like char",
			frontmatter: "a: \"",
			want:        map[string]string{"a": "\""},
		},
		{
			name:        "empty frontmatter",
			frontmatter: "",
			want:        map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseFrontmatterProperties(tt.frontmatter)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseFrontmatterProperties() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
