package cmd

import (
	"fmt"
	"os"

	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	Long: `Show current configuration settings.

Displays the effective configuration from environment variables,
config file, and token file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfig()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	configDir, _ := config.GetConfigDir()

	fmt.Println("Current Configuration")
	fmt.Println("=====================")
	fmt.Println()

	// Backend
	backend := string(cfg.Backend)
	if backend == "" {
		backend = "(not set, defaults to api)"
	}
	fmt.Printf("Backend:       %s\n", backend)

	// Token (masked)
	if cfg.Token != "" {
		masked := maskToken(cfg.Token)
		fmt.Printf("Token:         %s\n", masked)
	} else {
		fmt.Println("Token:         (not set)")
	}

	// Client ID (masked)
	if cfg.ClientID != "" {
		masked := maskToken(cfg.ClientID)
		fmt.Printf("Client ID:     %s\n", masked)
	} else {
		fmt.Println("Client ID:     (not set)")
	}

	// Client Secret (masked)
	if cfg.ClientSecret != "" {
		fmt.Println("Client Secret: (set)")
	} else {
		fmt.Println("Client Secret: (not set)")
	}

	fmt.Println()
	fmt.Println("Sources")
	fmt.Println("-------")

	// Check environment variables
	if os.Getenv("GOTION_BACKEND") != "" {
		fmt.Println("GOTION_BACKEND:        set")
	}
	if os.Getenv("GOTION_TOKEN") != "" {
		fmt.Println("GOTION_TOKEN:          set")
	}
	if os.Getenv("NOTION_TOKEN") != "" {
		fmt.Println("NOTION_TOKEN:          set")
	}
	if os.Getenv("GOTION_CLIENT_ID") != "" {
		fmt.Println("GOTION_CLIENT_ID:      set")
	}
	if os.Getenv("GOTION_CLIENT_SECRET") != "" {
		fmt.Println("GOTION_CLIENT_SECRET:  set")
	}

	// Check config file
	configPath := configDir + "/config.toml"
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config file:           %s\n", configPath)
	} else {
		fmt.Println("Config file:           (not found)")
	}

	// Check token file
	tokenPath := configDir + "/token.json"
	if _, err := os.Stat(tokenPath); err == nil {
		fmt.Printf("Token file:            %s\n", tokenPath)
	} else {
		fmt.Println("Token file:            (not found)")
	}

	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
