package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/longkey1/gotion/internal/gotion"
	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/spf13/cobra"
)

const (
	defaultCallbackPort    = 8080
	defaultMCPCallbackPort = 9998
	callbackTimeout        = 5 * time.Minute
)

type authOptions struct {
	port int
	mcp  bool
}

var authOpts = &authOptions{}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Notion API using OAuth",
	Long: `Authenticate with Notion API using OAuth.
This command initiates the OAuth flow to obtain and save access tokens.

Before running this command, you need to configure your OAuth credentials:
  - Set GOTION_CLIENT_ID and GOTION_CLIENT_SECRET environment variables
  - Or add client_id and client_secret to ~/.config/gotion/config.toml

Alternatively, use --mcp flag to authenticate via MCP OAuth (Dynamic Client Registration).
This does not require pre-configured client credentials.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuth(cmd.Context(), authOpts)
	},
}

func init() {
	authCmd.Flags().IntVarP(&authOpts.port, "port", "p", defaultCallbackPort, "Local callback server port")
	authCmd.Flags().BoolVar(&authOpts.mcp, "mcp", false, "Use MCP OAuth (Dynamic Client Registration)")
	rootCmd.AddCommand(authCmd)
}

func runAuth(ctx context.Context, opts *authOptions) error {
	// Check if token already exists
	configDir, _ := config.GetConfigDir()
	tokenPath := configDir + "/token.json"
	if _, err := os.Stat(tokenPath); err == nil {
		fmt.Printf("Token file already exists: %s\n", tokenPath)
		fmt.Print("Do you want to re-authenticate? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Use MCP OAuth only if --mcp flag is specified
	if opts.mcp {
		return runMCPAuth(ctx, opts)
	}

	// Traditional OAuth
	cfg, err := config.LoadOAuthConfig()
	if err != nil {
		return fmt.Errorf("failed to load OAuth config: %w", err)
	}
	return runTraditionalAuth(ctx, opts, cfg)
}

func runMCPAuth(ctx context.Context, opts *authOptions) error {
	port := defaultMCPCallbackPort
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	fmt.Println("Using MCP OAuth (Dynamic Client Registration)...")

	// Create MCP OAuth client
	mcpClient := gotion.NewMCPOAuthClient(callbackURL)

	// Step 1: Discover OAuth endpoints
	fmt.Println("Discovering OAuth endpoints...")
	if err := mcpClient.DiscoverEndpoints(ctx); err != nil {
		return fmt.Errorf("failed to discover endpoints: %w", err)
	}

	// Step 2: Register dynamic client
	fmt.Println("Registering dynamic client...")
	if err := mcpClient.RegisterClient(ctx); err != nil {
		return fmt.Errorf("failed to register client: %w", err)
	}
	fmt.Printf("Client registered: %s\n", mcpClient.GetClientID())

	// Step 3: Generate PKCE
	if err := mcpClient.GeneratePKCE(); err != nil {
		return fmt.Errorf("failed to generate PKCE: %w", err)
	}

	// Start callback server
	server, err := gotion.NewCallbackServer(port)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer server.Close()

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Get authorization URL
	authURL, err := mcpClient.GetAuthURL(state)
	if err != nil {
		return fmt.Errorf("failed to get auth URL: %w", err)
	}

	fmt.Println("Opening browser for Notion authorization...")
	fmt.Printf("If the browser doesn't open, visit this URL:\n%s\n\n", authURL)

	// Open browser
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	fmt.Println("Waiting for authorization...")

	// Wait for callback with timeout
	ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
	defer cancel()

	if err := server.Start(ctx, state); err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	code := server.Code()
	if code == "" {
		return fmt.Errorf("no authorization code received")
	}

	fmt.Println("Authorization received, exchanging code for token...")

	// Exchange code for token
	token, err := mcpClient.ExchangeCode(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// Save token with client_id for future refresh
	tokenData := &config.TokenData{
		AuthType:     config.AuthTypeMCP,
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		ClientID:     mcpClient.GetClientID(),
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
	}

	if err := config.SaveToken(tokenData); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("Authentication successful!")

	return nil
}

func runTraditionalAuth(ctx context.Context, opts *authOptions, cfg *config.Config) error {
	if err := cfg.ValidateOAuth(); err != nil {
		return err
	}

	port := opts.port

	// Start callback server
	server, err := gotion.NewCallbackServer(port)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer server.Close()

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", server.Port())

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Create OAuth client
	oauthClient := gotion.NewOAuthClient(&gotion.OAuthConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURI:  redirectURI,
	})

	// Get authorization URL
	authURL := oauthClient.GetAuthURL(state)

	fmt.Println("Opening browser for Notion authorization...")
	fmt.Printf("If the browser doesn't open, visit this URL:\n%s\n\n", authURL)

	// Open browser
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	fmt.Println("Waiting for authorization...")

	// Wait for callback with timeout
	ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
	defer cancel()

	if err := server.Start(ctx, state); err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	code := server.Code()
	if code == "" {
		return fmt.Errorf("no authorization code received")
	}

	fmt.Println("Authorization received, exchanging code for token...")

	// Exchange code for token
	token, err := oauthClient.ExchangeCode(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// Save token
	tokenData := &config.TokenData{
		AuthType:      config.AuthTypeAPI,
		AccessToken:   token.AccessToken,
		TokenType:     token.TokenType,
		BotID:         token.BotID,
		WorkspaceID:   token.WorkspaceID,
		WorkspaceName: token.WorkspaceName,
	}

	if err := config.SaveToken(tokenData); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("Authentication successful!")

	return nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}
