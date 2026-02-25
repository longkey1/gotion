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

	result, err := c.callTool(ctx, "notion-fetch", map[string]interface{}{
		"id": pageID,
	})
	if err != nil {
		return nil, err
	}

	title, url := extractPageMetadata(result.Result)

	return &types.PageResult{
		ID:      pageID,
		Title:   title,
		URL:     url,
		RawJSON: result.ContentJSON,
		Source:  "mcp",
	}, nil
}

// Search searches for pages using the MCP API
func (c *Client) Search(ctx context.Context, query string, opts *types.SearchOptions) (*types.SearchResult, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	args := map[string]interface{}{
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

func (c *Client) ensureInitialized(ctx context.Context) error {
	if c.initialized {
		return nil
	}

	params := map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
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

func (c *Client) callTool(ctx context.Context, name string, args map[string]interface{}) (*callToolResult, error) {
	params := map[string]interface{}{
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

func (c *Client) sendRequest(ctx context.Context, method string, params interface{}) (*jsonRPCResponse, error) {
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
	defer resp.Body.Close()

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
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int64       `json:"id"`
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
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Title    string                 `json:"title,omitempty"`
	URL      string                 `json:"url,omitempty"`
	Text     string                 `json:"text,omitempty"`
}

// FormatPage formats a page result as JSON string
func (c *Client) FormatPage(result *types.PageResult) (string, error) {
	return string(result.RawJSON), nil
}

// FormatSearch formats a search result as JSON string
func (c *Client) FormatSearch(result *types.SearchResult) (string, error) {
	return string(result.RawJSON), nil
}

// extractPageMetadata extracts title and url from MCP tool result content
func extractPageMetadata(result *toolResult) (title, url string) {
	if result == nil || len(result.Content) == 0 {
		return "", ""
	}

	for _, c := range result.Content {
		if c.Type == "text" {
			var resp mcpTextResponse
			if err := json.Unmarshal([]byte(c.Text), &resp); err == nil {
				return resp.Title, resp.URL
			}
		}
	}

	return "", ""
}
