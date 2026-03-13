package cmd

import (
	"context"
	"fmt"
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
		// Skip token refresh for non-API commands
		if skipTokenRefresh(cmd) {
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

// skipTokenRefresh returns true if the command should not trigger token refresh
func skipTokenRefresh(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "auth", "config", "version", "help", "completion":
			return true
		}
	}
	return false
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

	// Determine backend: check token data first, then config
	backend := tokenData.Backend
	if backend == "" {
		cfg, err := config.Load()
		if err != nil {
			return nil
		}
		backend = cfg.Backend
	}

	if backend != config.BackendMCP {
		return nil
	}

	// Refresh MCP token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	newToken, err := mcp.RefreshToken(ctx, tokenData.ClientID, tokenData.RefreshToken)
	if err != nil {
		// Re-read token file: another process may have already refreshed it
		reloaded, reloadErr := config.LoadToken()
		if reloadErr == nil && reloaded.AccessToken != tokenData.AccessToken {
			// Token was refreshed by another process, use it
			return nil
		}
		return fmt.Errorf("token refresh failed (re-authenticate with 'gotion auth'): %w", err)
	}

	// Update token data
	refreshedData := &config.TokenData{
		Backend:      config.BackendMCP,
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
