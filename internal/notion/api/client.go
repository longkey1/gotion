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

// GetPage retrieves a page by ID including its block children
func (c *Client) GetPage(ctx context.Context, pageID string, opts *types.GetPageOptions) (*types.PageResult, error) {
	pageID = normalizeID(pageID)

	// Fetch page metadata
	pageURL := fmt.Sprintf("%s/pages/%s", baseURL, pageID)

	if opts != nil && len(opts.FilterProperties) > 0 {
		pageURL += "?filter_properties=" + strings.Join(opts.FilterProperties, "&filter_properties=")
	}

	pageBody, err := c.doRequest(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	var page pageResponse
	if err := json.Unmarshal(pageBody, &page); err != nil {
		return nil, fmt.Errorf("failed to unmarshal page response: %w", err)
	}

	// Fetch all block children (with pagination)
	blocks, err := c.getAllBlockChildren(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get block children: %w", err)
	}

	// Combine page and blocks into a single response
	combinedResponse := map[string]interface{}{
		"page":   json.RawMessage(pageBody),
		"blocks": blocks,
	}

	combinedJSON, err := json.MarshalIndent(combinedResponse, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal combined response: %w", err)
	}

	title := extractTitle(page.Properties)
	properties := extractProperties(page.Properties)

	result := &types.PageResult{
		ID:      page.ID,
		URL:     page.URL,
		Title:   title,
		Props:   properties,
		RawJSON: combinedJSON,
		Source:  "api",
	}

	return result, nil
}

// getAllBlockChildren fetches all block children with pagination and recursively fetches nested children
func (c *Client) getAllBlockChildren(ctx context.Context, blockID string) ([]json.RawMessage, error) {
	var allBlocks []json.RawMessage
	var cursor string

	for {
		blocksURL := fmt.Sprintf("%s/blocks/%s/children", baseURL, blockID)
		if cursor != "" {
			blocksURL += "?start_cursor=" + cursor
		}

		body, err := c.doRequest(ctx, http.MethodGet, blocksURL, nil)
		if err != nil {
			return nil, err
		}

		var blocksResp blocksResponse
		if err := json.Unmarshal(body, &blocksResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal blocks response: %w", err)
		}

		// Process each block and recursively fetch children if needed
		for _, rawBlock := range blocksResp.Results {
			block, err := c.processBlockWithChildren(ctx, rawBlock)
			if err != nil {
				return nil, err
			}
			allBlocks = append(allBlocks, block)
		}

		if !blocksResp.HasMore {
			break
		}
		cursor = blocksResp.NextCursor
	}

	return allBlocks, nil
}

// processBlockWithChildren checks if a block has children and recursively fetches them
func (c *Client) processBlockWithChildren(ctx context.Context, rawBlock json.RawMessage) (json.RawMessage, error) {
	var block blockInfo
	if err := json.Unmarshal(rawBlock, &block); err != nil {
		return rawBlock, nil // Return as-is if we can't parse
	}

	if !block.HasChildren {
		return rawBlock, nil
	}

	// Fetch children recursively
	children, err := c.getAllBlockChildren(ctx, block.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch children for block %s: %w", block.ID, err)
	}

	// Parse the original block as a map to add children
	var blockMap map[string]interface{}
	if err := json.Unmarshal(rawBlock, &blockMap); err != nil {
		return rawBlock, nil
	}

	// Add children to the block
	blockMap["children"] = children

	// Re-marshal with children included
	enrichedBlock, err := json.Marshal(blockMap)
	if err != nil {
		return rawBlock, nil
	}

	return enrichedBlock, nil
}

// doRequest performs an HTTP request and returns the response body
func (c *Client) doRequest(ctx context.Context, method, url string, reqBody []byte) ([]byte, error) {
	var bodyReader io.Reader
	if reqBody != nil {
		bodyReader = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
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

	return body, nil
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

// FormatPage formats a page result as JSON string
func (c *Client) FormatPage(result *types.PageResult) (string, error) {
	return string(result.RawJSON), nil
}

// FormatSearch formats a search result as JSON string
func (c *Client) FormatSearch(result *types.SearchResult) (string, error) {
	return string(result.RawJSON), nil
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

type blocksResponse struct {
	Results    []json.RawMessage `json:"results"`
	NextCursor string            `json:"next_cursor"`
	HasMore    bool              `json:"has_more"`
}

type blockInfo struct {
	ID          string `json:"id"`
	HasChildren bool   `json:"has_children"`
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
