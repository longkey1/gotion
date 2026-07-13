package gotion

import "testing"

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
			want: "---\ntitle: \"My Page\"\nurl: https://www.notion.so/abc123\n---\n\n# Heading\n\nBody text.",
		},
		{
			name: "empty content omits body",
			output: &PageOutput{
				Title: "Empty",
				URL:   "https://www.notion.so/xyz",
			},
			want: "---\ntitle: \"Empty\"\nurl: https://www.notion.so/xyz\n---\n\n",
		},
		{
			name: "title with quotes is escaped",
			output: &PageOutput{
				Title: `He said "hi"`,
				URL:   "https://www.notion.so/q",
			},
			want: "---\ntitle: \"He said \\\"hi\\\"\"\nurl: https://www.notion.so/q\n---\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := FormatPage(tt.output); got != tt.want {
				t.Errorf("FormatPage() = %q, want %q", got, tt.want)
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
