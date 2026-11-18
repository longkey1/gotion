package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/longkey1/gotion/internal/notion/types"
)

const (
	mcpEndpoint = "https://mcp.notion.com/mcp"
)

// Client is a Notion MCP API client
type Client struct {
	httpClient  *http.Client
	accessToken string
	sessionID   string
	requestID   atomic.Int64
	initialized bool
}

// NewClient creates a new Notion MCP API client
func NewClient(token string) (*Client, error) {
	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		accessToken: token,
	}, nil
}

// GetPage retrieves a page by ID using the MCP API
func (c *Client) GetPage(ctx context.Context, pageID string, opts *types.GetPageOptions) (*types.PageResult, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	result, err := c.callTool(ctx, "notion-fetch", map[string]any{
		"id": pageID,
	})
	if err != nil {
		return nil, err
	}

	title, url, content, properties := extractPageMetadata(result.Result)

	return &types.PageResult{
		ID:         pageID,
		Title:      title,
		URL:        url,
		Content:    content,
		Properties: properties,
		RawJSON:    result.ContentJSON,
		Source:     "mcp",
	}, nil
}

// Search searches for pages using the MCP API
func (c *Client) Search(ctx context.Context, query string, opts *types.SearchOptions) (*types.SearchResult, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	args := map[string]any{
		"query": query,
	}

	if opts != nil && opts.PageSize > 0 {
		args["page_size"] = opts.PageSize
	}

	result, err := c.callTool(ctx, "notion-search", args)
	if err != nil {
		return nil, err
	}

	return &types.SearchResult{
		RawJSON: result.ContentJSON,
		Source:  "mcp",
	}, nil
}

// CreatePage creates a new page using the MCP API
func (c *Client) CreatePage(ctx context.Context, opts *types.CreatePageOptions) (*types.CreatePageResult, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	page := map[string]any{
		"properties": opts.Properties,
	}
	if opts.Content != "" {
		page["content"] = opts.Content
	}

	args := map[string]any{
		"pages": []any{page},
	}

	if opts.Parent != nil {
		parent := map[string]any{
			"type":           opts.Parent.Type,
			opts.Parent.Type: opts.Parent.ID,
		}
		args["parent"] = parent
	}

	result, err := c.callTool(ctx, "notion-create-pages", args)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	return &types.CreatePageResult{
		RawJSON: result.ContentJSON,
		Source:  "mcp",
	}, nil
}

// UpdatePage updates an existing page using the MCP API
func (c *Client) UpdatePage(ctx context.Context, pageID string, opts *types.UpdatePageOptions) (*types.UpdatePageResult, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	var lastResult *callToolResult

	if len(opts.Properties) > 0 {
		result, err := c.callTool(ctx, "notion-update-page", map[string]any{
			"page_id":    pageID,
			"command":    "update_properties",
			"properties": opts.Properties,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update properties: %w", err)
		}
		lastResult = result
	}

	if opts.Content != nil {
		result, err := c.callTool(ctx, "notion-update-page", map[string]any{
			"page_id": pageID,
			"command": "replace_content",
			"new_str": *opts.Content,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update content: %w", err)
		}
		lastResult = result
	}

	if lastResult == nil {
		return nil, fmt.Errorf("no properties or content to update")
	}

	return &types.UpdatePageResult{
		RawJSON: lastResult.ContentJSON,
		Source:  "mcp",
	}, nil
}

func (c *Client) ensureInitialized(ctx context.Context) error {
	if c.initialized {
		return nil
	}

	params := map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "gotion",
			"version": "0.1.0",
		},
	}

	resp, err := c.sendRequest(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	if errObj := resp.GetError(); errObj != nil {
		return fmt.Errorf("MCP initialize error: %s", errObj.Message)
	}

	// Send initialized notification
	_, err = c.sendRequest(ctx, "notifications/initialized", nil)
	if err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.initialized = true
	return nil
}

// callToolResult holds both parsed result and content JSON
type callToolResult struct {
	Result      *toolResult
	ContentJSON []byte
}

func (c *Client) callTool(ctx context.Context, name string, args map[string]any) (*callToolResult, error) {
	params := map[string]any{
		"name":      name,
		"arguments": args,
	}

	resp, err := c.sendRequest(ctx, "tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", name, err)
	}

	if errObj := resp.GetError(); errObj != nil {
		return nil, fmt.Errorf("MCP tool error: %s", errObj.Message)
	}

	var result toolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("MCP error: %s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("MCP error: unknown error")
	}

	// Marshal only content array
	contentJSON, err := json.MarshalIndent(result.Content, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %w", err)
	}

	return &callToolResult{
		Result:      &result,
		ContentJSON: contentJSON,
	}, nil
}

func (c *Client) sendRequest(ctx context.Context, method string, params any) (*jsonRPCResponse, error) {
	reqID := c.requestID.Add(1)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      reqID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, mcpEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+c.accessToken)

	if c.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Store session ID from response
	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		c.sessionID = sessionID
	}

	// Handle SSE response
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return c.parseSSEResponse(resp.Body, reqID)
	}

	// Handle JSON response
	var jsonResp jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &jsonResp, nil
}

func (c *Client) parseSSEResponse(body io.Reader, expectedID int64) (*jsonRPCResponse, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	var dataBuffer strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			dataBuffer.WriteString(data)
		} else if line == "" && dataBuffer.Len() > 0 {
			var resp jsonRPCResponse
			if err := json.Unmarshal([]byte(dataBuffer.String()), &resp); err != nil {
				dataBuffer.Reset()
				continue
			}

			if resp.ID == expectedID {
				return &resp, nil
			}

			dataBuffer.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read SSE response: %w", err)
	}

	return nil, fmt.Errorf("no response received for request ID %d", expectedID)
}

// Internal types

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      int64  `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
	ID      int64           `json:"id"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetError parses the error field which can be either a string or an object
func (r *jsonRPCResponse) GetError() *jsonRPCError {
	if len(r.Error) == 0 {
		return nil
	}

	// Try to unmarshal as object first
	var errObj jsonRPCError
	if err := json.Unmarshal(r.Error, &errObj); err == nil {
		return &errObj
	}

	// Try to unmarshal as string
	var errStr string
	if err := json.Unmarshal(r.Error, &errStr); err == nil {
		return &jsonRPCError{Message: errStr}
	}

	// Return raw error
	return &jsonRPCError{Message: string(r.Error)}
}

type toolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// mcpTextResponse represents the JSON structure in the text field
type mcpTextResponse struct {
	Metadata map[string]any `json:"metadata,omitempty"`
	Title    string         `json:"title,omitempty"`
	URL      string         `json:"url,omitempty"`
	Text     string         `json:"text,omitempty"`
}

// FormatPage formats a page result as JSON string
func (c *Client) FormatPage(result *types.PageResult) (string, error) {
	return string(result.RawJSON), nil
}

// FormatSearch formats a search result as JSON string
func (c *Client) FormatSearch(result *types.SearchResult) (string, error) {
	return string(result.RawJSON), nil
}

// extractPageMetadata extracts title, url, markdown content, and writable
// properties from MCP tool result content
func extractPageMetadata(result *toolResult) (title, url, content string, properties map[string]any) {
	if result == nil || len(result.Content) == 0 {
		return "", "", "", nil
	}

	for _, c := range result.Content {
		if c.Type == "text" {
			var resp mcpTextResponse
			if err := json.Unmarshal([]byte(c.Text), &resp); err == nil {
				return resp.Title, resp.URL, extractPageContent(resp.Text), extractWritableProperties(resp.Text, resp.Title)
			}
			// If not JSON, treat as plain markdown content
			if content == "" {
				content = c.Text
			}
		}
	}

	return title, url, content, nil
}

// extractWritableProperties returns the properties of a notion-fetch result
// that update_properties accepts back unchanged. Sending a read-only property
// (formula, created_time, unique_id, ...) makes Notion reject the whole update,
// so only keys that are columns of the parent data source schema and not
// listed as read-only are kept. The title column is dropped because the page
// title is carried separately. Pages without a data source schema yield nil.
func extractWritableProperties(text, title string) map[string]any {
	props := extractBlock(text, "properties")
	schema := extractBlock(text, "parent-data-source-schema")
	if props == "" || schema == "" {
		return nil
	}

	var all map[string]any
	if err := json.Unmarshal([]byte(props), &all); err != nil {
		return nil
	}

	columns := parseSchemaColumns(extractBlock(schema, "sqlite-table"))
	readOnly := parseReadOnlyColumns(schema)

	titleKey := ""
	for _, col := range columns {
		if v, ok := all[col].(string); ok && v == title {
			titleKey = col
			break
		}
	}

	writable := make(map[string]any)
	for _, col := range columns {
		v, ok := all[col]
		if !ok || col == "url" || col == titleKey || readOnly[col] {
			continue
		}
		writable[col] = v
	}

	if len(writable) == 0 {
		return nil
	}
	return writable
}

// extractBlock returns the text between a line-start "<tag ...>" line and
// its closing "</tag>" line, or "" when the block is absent.
func extractBlock(text, tag string) string {
	_, rest, found := strings.Cut(text, "\n<"+tag)
	if !found {
		return ""
	}
	// Skip attributes on the opening tag line
	_, rest, found = strings.Cut(rest, ">\n")
	if !found {
		return ""
	}
	body, _, found := strings.Cut(rest, "\n</"+tag+">")
	if !found {
		return ""
	}
	return body
}

// parseSchemaColumns returns column names, in order, from a CREATE TABLE
// statement where each column is on its own tab-indented line.
func parseSchemaColumns(table string) []string {
	var columns []string
	for line := range strings.Lines(table) {
		line, ok := strings.CutPrefix(line, "\t")
		if !ok {
			continue
		}
		if quoted, ok := strings.CutPrefix(line, `"`); ok {
			if name, ok := cutQuotedIdentifier(quoted); ok {
				columns = append(columns, name)
			}
			continue
		}
		if name, _, ok := strings.Cut(line, " "); ok {
			columns = append(columns, name)
		}
	}
	return columns
}

// cutQuotedIdentifier reads an SQL identifier after its opening quote,
// where a doubled quote stands for a literal one.
func cutQuotedIdentifier(s string) (string, bool) {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '"' {
			sb.WriteByte(s[i])
			continue
		}
		if i+1 < len(s) && s[i+1] == '"' {
			sb.WriteByte('"')
			i++
			continue
		}
		return sb.String(), true
	}
	return "", false
}

// parseReadOnlyColumns collects the quoted column names from the schema's
// read-only notice line, e.g.:
// These columns are read-only, so don't set them: "Created", "Seq".
func parseReadOnlyColumns(schema string) map[string]bool {
	readOnly := make(map[string]bool)
	before, _, _ := strings.Cut(schema, "<sqlite-table>")
	for line := range strings.Lines(before) {
		_, names, found := strings.Cut(line, "read-only")
		if !found {
			continue
		}
		for {
			_, rest, found := strings.Cut(names, `"`)
			if !found {
				break
			}
			name, rest, found := strings.Cut(rest, `"`)
			if !found {
				break
			}
			readOnly[name] = true
			names = rest
		}
	}
	return readOnly
}

// extractPageContent returns the page body inside the <content> block of a
// notion-fetch result. The fetch text also wraps the body with metadata
// (ancestor path, properties, data source schema); passing that back through
// replace_content would write it into the page. Text without a <content>
// block is returned unchanged.
func extractPageContent(text string) string {
	const openTag = "\n<content>"
	const closeTag = "\n</content>"

	// Match the opening tag at a line start: the preceding <properties> block is
	// single-line JSON, so a "<content>" string inside a property value cannot match.
	_, body, found := strings.Cut(text, openTag)
	if !found {
		return text
	}

	// The last closing tag ends the block, so a literal "</content>" line in the body is kept.
	body, _, found = strings.CutLast(body, closeTag)
	if !found {
		return text
	}

	return strings.TrimPrefix(body, "\n")
}
