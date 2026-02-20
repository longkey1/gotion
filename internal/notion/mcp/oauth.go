package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	serverURL          = "https://mcp.notion.com"
	defaultCallbackURL = "http://127.0.0.1:9998/callback"
)

// ProtectedResourceMetadata represents RFC 9728 metadata
type ProtectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
}

// AuthServerMetadata represents RFC 8414 metadata
type AuthServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// ClientRegistrationRequest represents RFC 7591 client registration request
type ClientRegistrationRequest struct {
	RedirectURIs                  []string `json:"redirect_uris"`
	TokenEndpointAuthMethod       string   `json:"token_endpoint_auth_method"`
	GrantTypes                    []string `json:"grant_types"`
	ResponseTypes                 []string `json:"response_types"`
	ClientName                    string   `json:"client_name"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}

// ClientRegistrationResponse represents RFC 7591 client registration response
type ClientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at,omitempty"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at,omitempty"`
	RedirectURIs            []string `json:"redirect_uris,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
}

// PKCEPair holds PKCE code_verifier and code_challenge
type PKCEPair struct {
	CodeVerifier  string
	CodeChallenge string
}

// OAuthToken represents the OAuth token response
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
}

// OAuthClient handles MCP OAuth operations with Dynamic Client Registration
type OAuthClient struct {
	httpClient   *http.Client
	mcpServerURL string
	callbackURL  string

	// Discovered metadata
	protectedResource *ProtectedResourceMetadata
	authServer        *AuthServerMetadata
	clientReg         *ClientRegistrationResponse

	// PKCE
	pkce *PKCEPair
}

// NewOAuthClient creates a new MCP OAuth client
func NewOAuthClient(callbackURL string) *OAuthClient {
	if callbackURL == "" {
		callbackURL = defaultCallbackURL
	}
	return &OAuthClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		mcpServerURL: serverURL,
		callbackURL:  callbackURL,
	}
}

// DiscoverEndpoints discovers OAuth endpoints using RFC 9728 and RFC 8414
func (c *OAuthClient) DiscoverEndpoints(ctx context.Context) error {
	// Step 1: Discover protected resource metadata (RFC 9728)
	prURL := c.mcpServerURL + "/.well-known/oauth-protected-resource"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, prURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create protected resource request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch protected resource metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch protected resource metadata: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var prMetadata ProtectedResourceMetadata
	if err := json.NewDecoder(resp.Body).Decode(&prMetadata); err != nil {
		return fmt.Errorf("failed to decode protected resource metadata: %w", err)
	}

	if len(prMetadata.AuthorizationServers) == 0 {
		return fmt.Errorf("no authorization servers found in protected resource metadata")
	}

	c.protectedResource = &prMetadata

	// Step 2: Discover auth server metadata (RFC 8414)
	authServerURL := prMetadata.AuthorizationServers[0]
	asURL := authServerURL + "/.well-known/oauth-authorization-server"
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, asURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create auth server request: %w", err)
	}

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch auth server metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch auth server metadata: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var asMetadata AuthServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&asMetadata); err != nil {
		return fmt.Errorf("failed to decode auth server metadata: %w", err)
	}

	if asMetadata.AuthorizationEndpoint == "" || asMetadata.TokenEndpoint == "" {
		return fmt.Errorf("missing required endpoints in auth server metadata")
	}

	c.authServer = &asMetadata

	return nil
}

// RegisterClient registers a dynamic OAuth client using RFC 7591
func (c *OAuthClient) RegisterClient(ctx context.Context) error {
	if c.authServer == nil {
		return fmt.Errorf("auth server metadata not discovered, call DiscoverEndpoints first")
	}

	if c.authServer.RegistrationEndpoint == "" {
		return fmt.Errorf("registration endpoint not available")
	}

	regReq := ClientRegistrationRequest{
		RedirectURIs:            []string{c.callbackURL},
		TokenEndpointAuthMethod: "none",
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		ClientName:              "gotion",
	}

	body, err := json.Marshal(regReq)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authServer.RegistrationEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register client: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to register client: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var regResp ClientRegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}

	if regResp.ClientID == "" {
		return fmt.Errorf("no client_id in registration response")
	}

	c.clientReg = &regResp

	return nil
}

// GeneratePKCE generates PKCE code_verifier and code_challenge (RFC 7636)
func (c *OAuthClient) GeneratePKCE() error {
	// Generate 32 random bytes for code_verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return fmt.Errorf("failed to generate random bytes: %w", err)
	}

	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate code_challenge = BASE64URL(SHA256(code_verifier))
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	c.pkce = &PKCEPair{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
	}

	return nil
}

// GetAuthURL returns the authorization URL with PKCE
func (c *OAuthClient) GetAuthURL(state string) (string, error) {
	if c.authServer == nil {
		return "", fmt.Errorf("auth server metadata not discovered")
	}
	if c.clientReg == nil {
		return "", fmt.Errorf("client not registered")
	}
	if c.pkce == nil {
		return "", fmt.Errorf("PKCE not generated")
	}

	params := url.Values{}
	params.Set("client_id", c.clientReg.ClientID)
	params.Set("redirect_uri", c.callbackURL)
	params.Set("response_type", "code")
	params.Set("code_challenge", c.pkce.CodeChallenge)
	params.Set("code_challenge_method", "S256")
	if state != "" {
		params.Set("state", state)
	}

	return c.authServer.AuthorizationEndpoint + "?" + params.Encode(), nil
}

// ExchangeCode exchanges an authorization code for an access token
func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	if c.authServer == nil {
		return nil, fmt.Errorf("auth server metadata not discovered")
	}
	if c.clientReg == nil {
		return nil, fmt.Errorf("client not registered")
	}
	if c.pkce == nil {
		return nil, fmt.Errorf("PKCE not generated")
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.callbackURL)
	data.Set("client_id", c.clientReg.ClientID)
	data.Set("code_verifier", c.pkce.CodeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authServer.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to exchange code: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in token response")
	}

	// Calculate expires_at if expires_in is provided
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Unix() + int64(token.ExpiresIn)
	}

	return &token, nil
}

// GetClientID returns the registered client ID
func (c *OAuthClient) GetClientID() string {
	if c.clientReg == nil {
		return ""
	}
	return c.clientReg.ClientID
}

// GetCallbackURL returns the callback URL
func (c *OAuthClient) GetCallbackURL() string {
	return c.callbackURL
}

// RefreshToken refreshes an access token using a refresh token
func RefreshToken(ctx context.Context, clientID, refreshToken string) (*OAuthToken, error) {
	client := &OAuthClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		mcpServerURL: serverURL,
	}

	// Discover endpoints
	if err := client.DiscoverEndpoints(ctx); err != nil {
		return nil, fmt.Errorf("failed to discover endpoints: %w", err)
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.authServer.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to refresh token: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in refresh response")
	}

	// Calculate expires_at if expires_in is provided
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Unix() + int64(token.ExpiresIn)
	}

	return &token, nil
}
