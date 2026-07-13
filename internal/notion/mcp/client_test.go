package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/longkey1/gotion/internal/notion/types"
)

func TestJSONRPCResponseGetError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errField string
		want     *jsonRPCError
	}{
		{
			name:     "no error",
			errField: "",
			want:     nil,
		},
		{
			name:     "error object",
			errField: `{"code": -32600, "message": "invalid request"}`,
			want:     &jsonRPCError{Code: -32600, Message: "invalid request"},
		},
		{
			name:     "error string",
			errField: `"something failed"`,
			want:     &jsonRPCError{Message: "something failed"},
		},
		{
			name:     "unparsable error falls back to raw",
			errField: `[1, 2]`,
			want:     &jsonRPCError{Message: "[1, 2]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &jsonRPCResponse{}
			if tt.errField != "" {
				resp.Error = json.RawMessage(tt.errField)
			}

			got := resp.GetError()
			if tt.want == nil {
				if got != nil {
					t.Fatalf("GetError() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("GetError() = nil, want error")
			}
			if got.Code != tt.want.Code || got.Message != tt.want.Message {
				t.Errorf("GetError() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestExtractPageMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		result      *toolResult
		wantTitle   string
		wantURL     string
		wantContent string
	}{
		{
			name:   "nil result",
			result: nil,
		},
		{
			name:   "empty content",
			result: &toolResult{},
		},
		{
			name: "JSON text content",
			result: &toolResult{
				Content: []toolContent{
					{Type: "text", Text: `{"title": "My Page", "url": "https://www.notion.so/abc", "text": "# Body"}`},
				},
			},
			wantTitle:   "My Page",
			wantURL:     "https://www.notion.so/abc",
			wantContent: "# Body",
		},
		{
			name: "plain markdown text content",
			result: &toolResult{
				Content: []toolContent{
					{Type: "text", Text: "# Just Markdown"},
				},
			},
			wantContent: "# Just Markdown",
		},
		{
			name: "non-text content is ignored",
			result: &toolResult{
				Content: []toolContent{
					{Type: "image"},
					{Type: "text", Text: "# After Image"},
				},
			},
			wantContent: "# After Image",
		},
		{
			name: "first plain text wins",
			result: &toolResult{
				Content: []toolContent{
					{Type: "text", Text: "first"},
					{Type: "text", Text: "second"},
				},
			},
			wantContent: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			title, url, content := extractPageMetadata(tt.result)
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
			if url != tt.wantURL {
				t.Errorf("url = %q, want %q", url, tt.wantURL)
			}
			if content != tt.wantContent {
				t.Errorf("content = %q, want %q", content, tt.wantContent)
			}
		})
	}
}

func TestParseSSEResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		expectedID int64
		wantResult string
		wantErr    string
	}{
		{
			name:       "single event",
			body:       "data: {\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {\"ok\": true}}\n\n",
			expectedID: 1,
			wantResult: `{"ok": true}`,
		},
		{
			name: "skips events with other IDs",
			body: "data: {\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {\"first\": true}}\n\n" +
				"data: {\"jsonrpc\": \"2.0\", \"id\": 2, \"result\": {\"second\": true}}\n\n",
			expectedID: 2,
			wantResult: `{"second": true}`,
		},
		{
			name: "skips malformed events",
			body: "data: not json\n\n" +
				"data: {\"jsonrpc\": \"2.0\", \"id\": 3, \"result\": {}}\n\n",
			expectedID: 3,
			wantResult: `{}`,
		},
		{
			name:       "no matching response",
			body:       "data: {\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {}}\n\n",
			expectedID: 99,
			wantErr:    "no response received for request ID 99",
		},
		{
			name:       "empty body",
			body:       "",
			expectedID: 1,
			wantErr:    "no response received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Client{}
			got, err := c.parseSSEResponse(strings.NewReader(tt.body), tt.expectedID)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseSSEResponse() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSSEResponse() error = %v", err)
			}
			if got.ID != tt.expectedID {
				t.Errorf("response ID = %d, want %d", got.ID, tt.expectedID)
			}
			if string(got.Result) != tt.wantResult {
				t.Errorf("result = %s, want %s", got.Result, tt.wantResult)
			}
		})
	}
}

func TestClientFormatPageAndSearch(t *testing.T) {
	t.Parallel()

	c, err := NewClient("token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	page, err := c.FormatPage(&types.PageResult{RawJSON: []byte(`[{"type": "text"}]`)})
	if err != nil {
		t.Fatalf("FormatPage() error = %v", err)
	}
	if page != `[{"type": "text"}]` {
		t.Errorf("FormatPage() = %q, want raw JSON", page)
	}

	search, err := c.FormatSearch(&types.SearchResult{RawJSON: []byte(`[]`)})
	if err != nil {
		t.Fatalf("FormatSearch() error = %v", err)
	}
	if search != `[]` {
		t.Errorf("FormatSearch() = %q, want raw JSON", search)
	}
}
