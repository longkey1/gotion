package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authURL  = "https://api.notion.com/v1/oauth/authorize"
	tokenURL = "https://api.notion.com/v1/oauth/token"
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

// Owner represents the owner of the token
type Owner struct {
	Type string `json:"type"`
	User *User  `json:"user,omitempty"`
}

// User represents a Notion user
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

	return authURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for an access token
func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.config.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
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
		var apiErr apiError
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
