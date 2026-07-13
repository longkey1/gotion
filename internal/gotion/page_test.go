package gotion

import "testing"

func TestExtractPageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "notion.so URL with hyphenated title and hex ID",
			input: "https://www.notion.so/workspace/My-Page-0123456789abcdef0123456789abcdef",
			want:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:  "notion.so URL with UUID format",
			input: "https://www.notion.so/01234567-89ab-cdef-0123-456789abcdef",
			want:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:  "notion.site URL",
			input: "https://example.notion.site/Page-0123456789abcdef0123456789abcdef",
			want:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:  "URL with query parameters",
			input: "https://www.notion.so/Page-0123456789abcdef0123456789abcdef?pvs=4",
			want:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:  "bare 32-character ID",
			input: "0123456789abcdef0123456789abcdef",
			want:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:  "bare UUID with hyphens",
			input: "01234567-89ab-cdef-0123-456789abcdef",
			want:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:  "notion URL without ID falls back to hyphen stripping",
			input: "https://www.notion.so/no-id-here",
			want:  "https://www.notion.so/noidhere",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ExtractPageID(tt.input); got != tt.want {
				t.Errorf("ExtractPageID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
