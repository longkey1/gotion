package gotion

import (
	"reflect"
	"strings"
	"testing"
)

func TestFormatPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output *PageOutput
		want   string
	}{
		{
			name: "title, url and content",
			output: &PageOutput{
				Title:   "My Page",
				URL:     "https://www.notion.so/abc123",
				Content: "# Heading\n\nBody text.",
			},
			want: "---\ntitle: My Page\nurl: https://www.notion.so/abc123\n---\n\n# Heading\n\nBody text.",
		},
		{
			name: "empty content omits body",
			output: &PageOutput{
				Title: "Empty",
				URL:   "https://www.notion.so/xyz",
			},
			want: "---\ntitle: Empty\nurl: https://www.notion.so/xyz\n---\n\n",
		},
		{
			name: "properties follow title and url in sorted order",
			output: &PageOutput{
				Title: "Task",
				URL:   "https://www.notion.so/t",
				Properties: map[string]any{
					"Status":               "Todo",
					"Tags":                 []any{"a", "b"},
					"date:Due:start":       "2026-12-01",
					"date:Due:is_datetime": float64(0),
				},
				Content: "body",
			},
			want: "---\ntitle: Task\nurl: https://www.notion.so/t\n" +
				"Status: Todo\nTags:\n  - a\n  - b\ndate:Due:is_datetime: 0\ndate:Due:start: \"2026-12-01\"\n" +
				"---\n\nbody",
		},
		{
			name: "title with quotes and colon is quoted",
			output: &PageOutput{
				Title: `He said "hi": yes`,
				URL:   "https://www.notion.so/q",
			},
			want: "---\ntitle: 'He said \"hi\": yes'\nurl: https://www.notion.so/q\n---\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FormatPage(tt.output)
			if err != nil {
				t.Fatalf("FormatPage() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("FormatPage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatPageRoundTrip(t *testing.T) {
	t.Parallel()

	titles := []string{
		"Plain",
		`He said "hi"`,
		`back\slash`,
		"key: value",
		"# not a comment",
		"2024-01-02",
		"true",
		"123",
		"日本語のタイトル",
	}

	for _, title := range titles {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			formatted, err := FormatPage(&PageOutput{
				Title:   title,
				URL:     "https://www.notion.so/abc",
				Content: "body",
			})
			if err != nil {
				t.Fatalf("FormatPage() error = %v", err)
			}

			parsed, err := ParseInput(strings.NewReader(formatted))
			if err != nil {
				t.Fatalf("ParseInput() error = %v", err)
			}

			want := &ParsedInput{
				Properties: map[string]any{"title": title},
				Content:    "body",
			}
			if !reflect.DeepEqual(parsed, want) {
				t.Errorf("round trip = %+v, want %+v", parsed, want)
			}
		})
	}
}

func TestFormatSearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output *SearchOutput
		want   string
	}{
		{
			name: "multiple pages",
			output: &SearchOutput{
				Pages: []SearchPageItem{
					{Title: "First", URL: "https://www.notion.so/1"},
					{Title: "Second", URL: "https://www.notion.so/2"},
				},
			},
			want: "- [First](https://www.notion.so/1)\n- [Second](https://www.notion.so/2)\n",
		},
		{
			name: "has more with cursor appends footer",
			output: &SearchOutput{
				Pages: []SearchPageItem{
					{Title: "Only", URL: "https://www.notion.so/1"},
				},
				HasMore:    true,
				NextCursor: "cursor-123",
			},
			want: "- [Only](https://www.notion.so/1)\n\n_More results available (cursor: cursor-123)_\n",
		},
		{
			name: "has more without cursor omits footer",
			output: &SearchOutput{
				Pages: []SearchPageItem{
					{Title: "Only", URL: "https://www.notion.so/1"},
				},
				HasMore: true,
			},
			want: "- [Only](https://www.notion.so/1)\n",
		},
		{
			name:   "no pages",
			output: &SearchOutput{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := FormatSearch(tt.output); got != tt.want {
				t.Errorf("FormatSearch() = %q, want %q", got, tt.want)
			}
		})
	}
}
