package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	// EnvPrefix is the prefix for environment variables
	EnvPrefix = "GOTION"

	// ConfigFileName is the name of the config file
	ConfigFileName = "config"
	// ConfigFileType is the type of the config file
	ConfigFileType = "toml"

	// TokenFileName is the name of the token file
	TokenFileName = "token.json"
)

// Config holds the application configuration
type Config struct {
	Token        string `mapstructure:"token"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// TokenData holds the OAuth token data
type TokenData struct {
	AccessToken   string `json:"access_token"`
	TokenType     string `json:"token_type"`
	BotID         string `json:"bot_id"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
}

// Load loads configuration from environment variables and config file
func Load() (*Config, error) {
	v := viper.New()

	// Set up environment variable binding
	v.SetEnvPrefix(EnvPrefix)
	v.AutomaticEnv()

	// Check GOTION_TOKEN first
	if token := os.Getenv("GOTION_TOKEN"); token != "" {
		return &Config{Token: token}, nil
	}

	// Then check NOTION_TOKEN
	if token := os.Getenv("NOTION_TOKEN"); token != "" {
		return &Config{Token: token}, nil
	}

	// Try to load OAuth token from token file
	tokenData, err := LoadToken()
	if err == nil && tokenData.AccessToken != "" {
		cfg := &Config{Token: tokenData.AccessToken}
		// Also load OAuth settings from config file for potential refresh
		loadOAuthSettings(cfg)
		return cfg, nil
	}

	// Try to load from config file
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(configDir)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, return empty config
		return &Config{}, nil
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// LoadOAuthConfig loads OAuth-specific configuration
func LoadOAuthConfig() (*Config, error) {
	v := viper.New()

	// Check environment variables first
	clientID := os.Getenv("GOTION_CLIENT_ID")
	clientSecret := os.Getenv("GOTION_CLIENT_SECRET")

	if clientID != "" && clientSecret != "" {
		return &Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		}, nil
	}

	// Try to load from config file
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(configDir)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		return &Config{}, nil
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// loadOAuthSettings loads OAuth settings into config
func loadOAuthSettings(cfg *Config) {
	v := viper.New()

	configDir, err := GetConfigDir()
	if err != nil {
		return
	}

	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(configDir)

	if err := v.ReadInConfig(); err != nil {
		return
	}

	cfg.ClientID = v.GetString("client_id")
	cfg.ClientSecret = v.GetString("client_secret")
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "gotion"), nil
}

// EnsureConfigDir ensures the configuration directory exists
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(configDir, 0700)
}

// SaveToken saves the OAuth token to the token file
func SaveToken(token *TokenData) error {
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	tokenPath := filepath.Join(configDir, TokenFileName)

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadToken loads the OAuth token from the token file
func LoadToken() (*TokenData, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	tokenPath := filepath.Join(configDir, TokenFileName)

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}

	var token TokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// DeleteToken deletes the OAuth token file
func DeleteToken() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	tokenPath := filepath.Join(configDir, TokenFileName)

	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("token is required. Run 'gotion auth login' or set GOTION_TOKEN/NOTION_TOKEN environment variable")
	}
	return nil
}

// ValidateOAuth checks if the OAuth configuration is valid
func (c *Config) ValidateOAuth() error {
	if c.ClientID == "" {
		return fmt.Errorf("client_id is required. Set GOTION_CLIENT_ID environment variable or configure in ~/.config/gotion/config.toml")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("client_secret is required. Set GOTION_CLIENT_SECRET environment variable or configure in ~/.config/gotion/config.toml")
	}
	return nil
}
