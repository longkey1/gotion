package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/longkey1/gotion/internal/notion/mcp"
	"github.com/spf13/cobra"
)

// writeAnnotation marks a command as mutating Notion state (see create.go/update.go).
const writeAnnotation = "write"

var rootCmd = &cobra.Command{
	Use:   "gotion",
	Short: "A CLI tool for Notion API",
	Long:  `gotion is a command-line interface for interacting with the Notion API.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if isWriteCommand(cmd) {
			if err := checkReadOnly(); err != nil {
				return err
			}
		}
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

// isWriteCommand returns true if cmd (or an ancestor) is annotated as a write command.
func isWriteCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[writeAnnotation] == "true" {
			return true
		}
	}
	return false
}

// checkReadOnly returns an error if read-only mode is enabled.
func checkReadOnly() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.ReadOnly {
		return fmt.Errorf("read-only mode is enabled (read_only/GOTION_READ_ONLY); write commands are disabled")
	}
	return nil
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Serialize refresh across processes so the same refresh token is never used twice
	unlock, err := config.LockToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire token lock: %w", err)
	}
	defer unlock()

	// Re-read under the lock: another process may have refreshed while we waited
	tokenData, err = config.LoadToken()
	if err != nil {
		return nil
	}
	if !tokenData.NeedsRefresh() {
		return nil
	}

	// Refresh MCP token
	newToken, err := mcp.RefreshToken(ctx, tokenData.ClientID, tokenData.RefreshToken)
	if err != nil {
		if oauthErr, ok := errors.AsType[*mcp.OAuthError](err); ok && oauthErr.IsInvalidGrant() {
			return fmt.Errorf("token refresh failed: the refresh token has expired or been revoked — Notion refresh tokens expire 180 days after the initial authorization (even with regular use) or after 30 days without use; re-authenticate with 'gotion auth': %w", err)
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
