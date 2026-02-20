package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/longkey1/gotion/internal/gotion"
	"github.com/longkey1/gotion/internal/notion/types"
)

const (
	baseURL       = "https://api.notion.com/v1"
	notionVersion = "2022-06-28"
)

// Client is a Notion REST API client
type Client struct {
	httpClient *http.Client
	token      string
}

// NewClient creates a new Notion REST API client
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{},
		token:      token,
	}
}

// GetPage retrieves a page by ID
func (c *Client) GetPage(ctx context.Context, pageID string, opts *types.GetPageOptions) (*types.PageResult, error) {
	pageID = normalizeID(pageID)

	url := fmt.Sprintf("%s/pages/%s", baseURL, pageID)

	if opts != nil && len(opts.FilterProperties) > 0 {
		url += "?filter_properties=" + strings.Join(opts.FilterProperties, "&filter_properties=")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr apiError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, &apiErr
	}

	var page pageResponse
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	title := extractTitle(page.Properties)
	properties := extractProperties(page.Properties)

	result := &types.PageResult{
		ID:      page.ID,
		URL:     page.URL,
		Title:   title,
		Props:   properties,
		RawJSON: body,
		Source:  "api",
	}

	return result, nil
}

// Search searches for pages
func (c *Client) Search(ctx context.Context, query string, opts *types.SearchOptions) (*types.SearchResult, error) {
	url := fmt.Sprintf("%s/search", baseURL)

	searchReq := searchRequest{
		Query: query,
		Filter: &searchFilter{
			Value:    "page",
			Property: "object",
		},
	}

	if opts != nil {
		if opts.PageSize > 0 {
			searchReq.PageSize = opts.PageSize
		}
		if opts.StartCursor != "" {
			searchReq.StartCursor = opts.StartCursor
		}
		if opts.Sort != "" {
			searchReq.Sort = &searchSort{
				Direction: opts.Sort,
				Timestamp: "last_edited_time",
			}
		}
	}

	jsonBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr apiError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, &apiErr
	}

	var searchResp searchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var pages []types.PageSummary
	for _, page := range searchResp.Results {
		title := extractTitle(page.Properties)
		pages = append(pages, types.PageSummary{
			ID:    page.ID,
			Title: title,
			URL:   page.URL,
		})
	}

	result := &types.SearchResult{
		Pages:      pages,
		HasMore:    searchResp.HasMore,
		NextCursor: searchResp.NextCursor,
		RawJSON:    body,
		Source:     "api",
	}

	return result, nil
}

// ToPageOutput converts PageResult to the intermediate PageOutput structure
func (c *Client) ToPageOutput(result *types.PageResult) *gotion.PageOutput {
	// Build content from properties
	var content strings.Builder
	for name, value := range result.Props {
		if name == "title" {
			continue
		}
		content.WriteString(fmt.Sprintf("- **%s:** %s\n", name, value))
	}

	return &gotion.PageOutput{
		Title:   result.Title,
		URL:     result.URL,
		Content: content.String(),
	}
}

// ToSearchOutput converts SearchResult to the intermediate SearchOutput structure
func (c *Client) ToSearchOutput(result *types.SearchResult) *gotion.SearchOutput {
	pages := make([]gotion.SearchPageItem, len(result.Pages))
	for i, p := range result.Pages {
		pages[i] = gotion.SearchPageItem{
			Title: p.Title,
			URL:   p.URL,
		}
	}

	return &gotion.SearchOutput{
		Pages:      pages,
		HasMore:    result.HasMore,
		NextCursor: result.NextCursor,
	}
}

// FormatPage formats a page result
func (c *Client) FormatPage(result *types.PageResult, format types.OutputFormat) (string, error) {
	switch format {
	case types.FormatJSON:
		return string(result.RawJSON), nil
	case types.FormatMarkdown, "":
		return gotion.FormatPage(c.ToPageOutput(result)), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatSearch formats a search result
func (c *Client) FormatSearch(result *types.SearchResult, format types.OutputFormat) (string, error) {
	switch format {
	case types.FormatJSON:
		return string(result.RawJSON), nil
	case types.FormatMarkdown, "":
		return gotion.FormatSearch(c.ToSearchOutput(result)), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionVersion)
}

func normalizeID(id string) string {
	return strings.ReplaceAll(id, "-", "")
}

// Internal types for API responses

type pageResponse struct {
	ID         string              `json:"id"`
	URL        string              `json:"url"`
	Properties map[string]property `json:"properties"`
}

type property struct {
	Type     string     `json:"type"`
	Title    []richText `json:"title,omitempty"`
	RichText []richText `json:"rich_text,omitempty"`
}

type richText struct {
	PlainText string `json:"plain_text"`
}

type searchRequest struct {
	Query       string        `json:"query,omitempty"`
	Sort        *searchSort   `json:"sort,omitempty"`
	Filter      *searchFilter `json:"filter,omitempty"`
	StartCursor string        `json:"start_cursor,omitempty"`
	PageSize    int           `json:"page_size,omitempty"`
}

type searchSort struct {
	Direction string `json:"direction"`
	Timestamp string `json:"timestamp"`
}

type searchFilter struct {
	Value    string `json:"value"`
	Property string `json:"property"`
}

type searchResponse struct {
	Results    []pageResponse `json:"results"`
	NextCursor string         `json:"next_cursor"`
	HasMore    bool           `json:"has_more"`
}

type apiError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *apiError) Error() string {
	return e.Message
}

func extractTitle(props map[string]property) string {
	for _, prop := range props {
		if prop.Type == "title" && len(prop.Title) > 0 {
			var sb strings.Builder
			for _, text := range prop.Title {
				sb.WriteString(text.PlainText)
			}
			return sb.String()
		}
	}
	return ""
}

func extractProperties(props map[string]property) map[string]string {
	result := make(map[string]string)
	for name, prop := range props {
		switch prop.Type {
		case "title":
			if len(prop.Title) > 0 {
				var sb strings.Builder
				for _, text := range prop.Title {
					sb.WriteString(text.PlainText)
				}
				result[name] = sb.String()
			}
		case "rich_text":
			if len(prop.RichText) > 0 {
				var sb strings.Builder
				for _, text := range prop.RichText {
					sb.WriteString(text.PlainText)
				}
				result[name] = sb.String()
			}
		}
	}
	return result
}
