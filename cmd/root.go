package cmd

import (
	"context"
	"time"

	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/longkey1/gotion/internal/notion/mcp"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gotion",
	Short: "A CLI tool for Notion API",
	Long:  `gotion is a command-line interface for interacting with the Notion API.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip token refresh for auth and config commands
		if cmd.Name() == "auth" || cmd.Name() == "config" || cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}
		return refreshTokenIfNeeded()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here if needed
}

// refreshTokenIfNeeded checks and refreshes the token if expired
func refreshTokenIfNeeded() error {
	tokenData, err := config.LoadToken()
	if err != nil {
		// No token file, skip refresh
		return nil
	}

	if !tokenData.NeedsRefresh() {
		return nil
	}

	// Only MCP tokens support refresh
	cfg, err := config.Load()
	if err != nil {
		return nil
	}

	if cfg.Backend != config.BackendMCP {
		return nil
	}

	// Refresh MCP token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	newToken, err := mcp.RefreshToken(ctx, tokenData.ClientID, tokenData.RefreshToken)
	if err != nil {
		// Refresh failed, continue with existing token
		return nil
	}

	// Update token data
	refreshedData := &config.TokenData{
		Backend:      tokenData.Backend,
		AccessToken:  newToken.AccessToken,
		TokenType:    newToken.TokenType,
		ClientID:     tokenData.ClientID,
		RefreshToken: newToken.RefreshToken,
		ExpiresAt:    newToken.ExpiresAt,
	}

	// Keep refresh token if new one is not provided
	if refreshedData.RefreshToken == "" {
		refreshedData.RefreshToken = tokenData.RefreshToken
	}

	// Save the refreshed token
	return config.SaveToken(refreshedData)
}
