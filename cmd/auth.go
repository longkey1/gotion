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
	defaultCallbackPort = 8080
	callbackTimeout     = 5 * time.Minute
)

type authOptions struct {
	port int
}

var authOpts = &authOptions{}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Notion API using OAuth",
	Long: `Authenticate with Notion API using OAuth.
This command initiates the OAuth flow to obtain and save access tokens.

Before running this command, you need to configure your OAuth credentials:
  - Set GOTION_CLIENT_ID and GOTION_CLIENT_SECRET environment variables
  - Or add client_id and client_secret to ~/.config/gotion/config.toml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuth(cmd.Context(), authOpts)
	},
}

func init() {
	authCmd.Flags().IntVarP(&authOpts.port, "port", "p", defaultCallbackPort, "Local callback server port")
	rootCmd.AddCommand(authCmd)
}

func runAuth(ctx context.Context, opts *authOptions) error {
	// Load OAuth configuration
	cfg, err := config.LoadOAuthConfig()
	if err != nil {
		return fmt.Errorf("failed to load OAuth config: %w", err)
	}

	if err := cfg.ValidateOAuth(); err != nil {
		return err
	}

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

	// Start callback server
	server, err := gotion.NewCallbackServer(opts.port)
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
