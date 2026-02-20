package gotion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// NotionAuthURL is the Notion OAuth authorization endpoint
	NotionAuthURL = "https://api.notion.com/v1/oauth/authorize"
	// NotionTokenURL is the Notion OAuth token endpoint
	NotionTokenURL = "https://api.notion.com/v1/oauth/token"
)

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// OAuthToken represents the OAuth token response
type OAuthToken struct {
	AccessToken          string    `json:"access_token"`
	TokenType            string    `json:"token_type"`
	BotID                string    `json:"bot_id"`
	WorkspaceID          string    `json:"workspace_id"`
	WorkspaceName        string    `json:"workspace_name"`
	WorkspaceIcon        string    `json:"workspace_icon"`
	DuplicatedTemplateID string    `json:"duplicated_template_id,omitempty"`
	Owner                *Owner    `json:"owner,omitempty"`
	ExpiresAt            time.Time `json:"-"`
}

// OAuthClient handles OAuth operations
type OAuthClient struct {
	config     *OAuthConfig
	httpClient *http.Client
}

// NewOAuthClient creates a new OAuth client
func NewOAuthClient(config *OAuthConfig) *OAuthClient {
	return &OAuthClient{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAuthURL returns the authorization URL
func (c *OAuthClient) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", c.config.ClientID)
	params.Set("redirect_uri", c.config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("owner", "user")
	if state != "" {
		params.Set("state", state)
	}

	return NotionAuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for an access token
func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.config.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, NotionTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Basic authentication with client_id:client_secret
	auth := base64.StdEncoding.EncodeToString([]byte(c.config.ClientID + ":" + c.config.ClientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return nil, fmt.Errorf("OAuth error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, &apiErr
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// CallbackServer handles the OAuth callback
type CallbackServer struct {
	port     int
	listener net.Listener
	code     string
	state    string
	err      error
	done     chan struct{}
}

// NewCallbackServer creates a new callback server
func NewCallbackServer(port int) (*CallbackServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	return &CallbackServer{
		port:     port,
		listener: listener,
		done:     make(chan struct{}),
	}, nil
}

// Port returns the actual port the server is listening on
func (s *CallbackServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// Start starts the callback server and waits for the callback
func (s *CallbackServer) Start(ctx context.Context, expectedState string) error {
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			query := r.URL.Query()

			// Check for error
			if errCode := query.Get("error"); errCode != "" {
				s.err = fmt.Errorf("OAuth error: %s", errCode)
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<html><body><h1>Authentication Failed</h1><p>%s</p><p>You can close this window.</p></body></html>`, errCode)
				close(s.done)
				return
			}

			// Verify state
			state := query.Get("state")
			if expectedState != "" && state != expectedState {
				s.err = fmt.Errorf("state mismatch")
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><body><h1>Authentication Failed</h1><p>State mismatch</p><p>You can close this window.</p></body></html>`)
				close(s.done)
				return
			}

			// Get authorization code
			code := query.Get("code")
			if code == "" {
				s.err = fmt.Errorf("no authorization code received")
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><body><h1>Authentication Failed</h1><p>No authorization code received</p><p>You can close this window.</p></body></html>`)
				close(s.done)
				return
			}

			s.code = code
			s.state = state
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h1>Authentication Successful!</h1><p>You can close this window and return to the terminal.</p></body></html>`)
			close(s.done)
		}),
	}

	go func() {
		_ = server.Serve(s.listener)
	}()

	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return ctx.Err()
	case <-s.done:
		_ = server.Shutdown(context.Background())
		return s.err
	}
}

// Code returns the authorization code received
func (s *CallbackServer) Code() string {
	return s.code
}

// Close closes the callback server
func (s *CallbackServer) Close() error {
	return s.listener.Close()
}
