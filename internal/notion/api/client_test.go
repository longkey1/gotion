package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/longkey1/gotion/internal/gotion"
	"github.com/longkey1/gotion/internal/notion/types"
)

func TestNormalizeID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "UUID with hyphens",
			input: "01234567-89ab-cdef-0123-456789abcdef",
			want:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:  "already normalized",
			input: "0123456789abcdef0123456789abcdef",
			want:  "0123456789abcdef0123456789abcdef",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeID(tt.input); got != tt.want {
				t.Errorf("normalizeID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		props map[string]property
		want  string
	}{
		{
			name: "single title fragment",
			props: map[string]property{
				"Name": {Type: "title", Title: []richText{{PlainText: "My Page"}}},
			},
			want: "My Page",
		},
		{
			name: "multiple title fragments are concatenated",
			props: map[string]property{
				"Name": {Type: "title", Title: []richText{{PlainText: "My "}, {PlainText: "Page"}}},
			},
			want: "My Page",
		},
		{
			name: "no title property",
			props: map[string]property{
				"Desc": {Type: "rich_text", RichText: []richText{{PlainText: "text"}}},
			},
			want: "",
		},
		{
			name: "title property with empty fragments",
			props: map[string]property{
				"Name": {Type: "title"},
			},
			want: "",
		},
		{
			name:  "empty properties",
			props: map[string]property{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := extractTitle(tt.props); got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		props map[string]property
		want  map[string]string
	}{
		{
			name: "title and rich text",
			props: map[string]property{
				"Name": {Type: "title", Title: []richText{{PlainText: "My Page"}}},
				"Desc": {Type: "rich_text", RichText: []richText{{PlainText: "Hello "}, {PlainText: "world"}}},
			},
			want: map[string]string{"Name": "My Page", "Desc": "Hello world"},
		},
		{
			name: "unsupported property types are skipped",
			props: map[string]property{
				"Status": {Type: "select"},
				"Name":   {Type: "title", Title: []richText{{PlainText: "T"}}},
			},
			want: map[string]string{"Name": "T"},
		},
		{
			name: "empty fragments are skipped",
			props: map[string]property{
				"Name": {Type: "title"},
				"Desc": {Type: "rich_text"},
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractProperties(tt.props)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractProperties() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestToPageOutput(t *testing.T) {
	t.Parallel()

	c := NewClient("token")
	result := &types.PageResult{
		Title: "My Page",
		URL:   "https://www.notion.so/abc",
		Props: map[string]string{
			"title": "My Page",
			"Desc":  "hello",
		},
	}

	got := c.ToPageOutput(result)

	if got.Title != "My Page" {
		t.Errorf("Title = %q, want %q", got.Title, "My Page")
	}
	if got.URL != "https://www.notion.so/abc" {
		t.Errorf("URL = %q, want %q", got.URL, "https://www.notion.so/abc")
	}
	if want := "- **Desc:** hello\n"; got.Content != want {
		t.Errorf("Content = %q, want %q (title property must be excluded)", got.Content, want)
	}
}

func TestToSearchOutput(t *testing.T) {
	t.Parallel()

	c := NewClient("token")
	result := &types.SearchResult{
		Pages: []types.PageSummary{
			{ID: "1", Title: "First", URL: "https://www.notion.so/1"},
			{ID: "2", Title: "Second", URL: "https://www.notion.so/2"},
		},
		HasMore:    true,
		NextCursor: "cur",
	}

	got := c.ToSearchOutput(result)

	want := &gotion.SearchOutput{
		Pages: []gotion.SearchPageItem{
			{Title: "First", URL: "https://www.notion.so/1"},
			{Title: "Second", URL: "https://www.notion.so/2"},
		},
		HasMore:    true,
		NextCursor: "cur",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ToSearchOutput() = %+v, want %+v", got, want)
	}
}

func TestDoRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		response   string
		want       string
		wantErr    string
		wantAPIErr bool
	}{
		{
			name:     "success returns body",
			status:   http.StatusOK,
			response: `{"ok": true}`,
			want:     `{"ok": true}`,
		},
		{
			name:       "API error with JSON body",
			status:     http.StatusNotFound,
			response:   `{"status": 404, "code": "object_not_found", "message": "Could not find page"}`,
			wantErr:    "Could not find page",
			wantAPIErr: true,
		},
		{
			name:     "API error with non-JSON body",
			status:   http.StatusBadGateway,
			response: "bad gateway",
			wantErr:  "API error (status 502): bad gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotAuth, gotVersion string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				gotVersion = r.Header.Get("Notion-Version")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			c := NewClient("secret-token")
			body, err := c.doRequest(context.Background(), http.MethodGet, srv.URL, nil)

			if gotAuth != "Bearer secret-token" {
				t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer secret-token")
			}
			if gotVersion != notionVersion {
				t.Errorf("Notion-Version header = %q, want %q", gotVersion, notionVersion)
			}

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("doRequest() error = %v, want error containing %q", err, tt.wantErr)
				}
				if tt.wantAPIErr {
					if _, ok := err.(*apiError); !ok {
						t.Errorf("doRequest() error type = %T, want *apiError", err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("doRequest() error = %v", err)
			}
			if string(body) != tt.want {
				t.Errorf("doRequest() body = %q, want %q", body, tt.want)
			}
		})
	}
}

func TestCreateAndUpdatePageUnsupported(t *testing.T) {
	t.Parallel()

	c := NewClient("token")
	ctx := context.Background()

	if _, err := c.CreatePage(ctx, &types.CreatePageOptions{}); err == nil {
		t.Error("CreatePage() error = nil, want unsupported error")
	}
	if _, err := c.UpdatePage(ctx, "id", &types.UpdatePageOptions{}); err == nil {
		t.Error("UpdatePage() error = nil, want unsupported error")
	}
}

func TestFormatPageAndSearch(t *testing.T) {
	t.Parallel()

	c := NewClient("token")

	page, err := c.FormatPage(&types.PageResult{RawJSON: []byte(`{"page": 1}`)})
	if err != nil {
		t.Fatalf("FormatPage() error = %v", err)
	}
	if page != `{"page": 1}` {
		t.Errorf("FormatPage() = %q, want raw JSON", page)
	}

	search, err := c.FormatSearch(&types.SearchResult{RawJSON: []byte(`{"results": []}`)})
	if err != nil {
		t.Fatalf("FormatSearch() error = %v", err)
	}
	if search != `{"results": []}` {
		t.Errorf("FormatSearch() = %q, want raw JSON", search)
	}
}

func TestAPIErrorError(t *testing.T) {
	t.Parallel()

	err := &apiError{Status: 400, Code: "validation_error", Message: "bad request"}
	if got := err.Error(); got != "bad request" {
		t.Errorf("Error() = %q, want %q", got, "bad request")
	}
}
