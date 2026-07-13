package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setHome redirects the user home directory (and thus GetConfigDir) to a
// temp directory and clears all configuration environment variables so
// tests are isolated from the host environment. Tests using it cannot be
// parallel because it mutates the process environment.
func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range []string{
		"GOTION_BACKEND",
		"GOTION_API_TOKEN",
		"GOTION_API_CLIENT_ID",
		"GOTION_API_CLIENT_SECRET",
		"NOTION_TOKEN",
	} {
		t.Setenv(key, "")
	}
	return home
}

// writeConfigFile writes content to ~/.config/gotion/config.toml under
// the redirected home directory.
func writeConfigFile(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "gotion")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		env     map[string]string
		token   *TokenData
		want    Config
		wantErr bool
	}{
		{
			name: "no config file and no env",
			want: Config{},
		},
		{
			name: "config file values",
			config: `backend = "mcp"
api_token = "file-token"
api_client_id = "file-client"
api_client_secret = "file-secret"
`,
			want: Config{
				Backend:      BackendMCP,
				Token:        "file-token",
				ClientID:     "file-client",
				ClientSecret: "file-secret",
			},
		},
		{
			name: "environment variables override config file",
			config: `backend = "api"
api_token = "file-token"
`,
			env: map[string]string{
				"GOTION_BACKEND":   "mcp",
				"GOTION_API_TOKEN": "env-token",
			},
			want: Config{Backend: BackendMCP, Token: "env-token"},
		},
		{
			name: "NOTION_TOKEN is a fallback for missing token",
			env: map[string]string{
				"NOTION_TOKEN": "notion-token",
			},
			want: Config{Token: "notion-token"},
		},
		{
			name: "GOTION_API_TOKEN wins over NOTION_TOKEN",
			env: map[string]string{
				"GOTION_API_TOKEN": "gotion-token",
				"NOTION_TOKEN":     "notion-token",
			},
			want: Config{Token: "gotion-token"},
		},
		{
			name: "token file is the last fallback",
			token: &TokenData{
				Backend:     BackendMCP,
				AccessToken: "saved-token",
				ClientID:    "saved-client",
			},
			want: Config{Token: "saved-token", ClientID: "saved-client"},
		},
		{
			name:   "config token wins over token file",
			config: "api_token = \"file-token\"\n",
			token: &TokenData{
				AccessToken: "saved-token",
			},
			want: Config{Token: "file-token"},
		},
		{
			name:    "broken config file",
			config:  "backend = [unclosed\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := setHome(t)
			if tt.config != "" {
				writeConfigFile(t, home, tt.config)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if tt.token != nil {
				if err := SaveToken(tt.token); err != nil {
					t.Fatal(err)
				}
			}

			got, err := Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if *got != tt.want {
				t.Errorf("Load() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestLoadOAuthConfig(t *testing.T) {
	home := setHome(t)
	writeConfigFile(t, home, `api_client_id = "file-client"
api_client_secret = "file-secret"
`)
	t.Setenv("GOTION_API_CLIENT_ID", "env-client")

	got, err := LoadOAuthConfig()
	if err != nil {
		t.Fatalf("LoadOAuthConfig() error = %v", err)
	}
	if got.ClientID != "env-client" {
		t.Errorf("ClientID = %q, want %q (env overrides file)", got.ClientID, "env-client")
	}
	if got.ClientSecret != "file-secret" {
		t.Errorf("ClientSecret = %q, want %q", got.ClientSecret, "file-secret")
	}
}

func TestTokenDataIsTokenExpired(t *testing.T) {
	t.Parallel()

	now := time.Now().Unix()

	tests := []struct {
		name  string
		token TokenData
		want  bool
	}{
		{
			name:  "no expiration info is treated as valid",
			token: TokenData{},
			want:  false,
		},
		{
			name:  "far future expiry",
			token: TokenData{ExpiresAt: now + 3600},
			want:  false,
		},
		{
			name:  "already expired",
			token: TokenData{ExpiresAt: now - 10},
			want:  true,
		},
		{
			name:  "within five minute margin",
			token: TokenData{ExpiresAt: now + 60},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.token.IsTokenExpired(); got != tt.want {
				t.Errorf("IsTokenExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenDataNeedsRefresh(t *testing.T) {
	t.Parallel()

	now := time.Now().Unix()

	tests := []struct {
		name  string
		token TokenData
		want  bool
	}{
		{
			name:  "expired with refresh token",
			token: TokenData{RefreshToken: "r", ExpiresAt: now - 10},
			want:  true,
		},
		{
			name:  "expired without refresh token",
			token: TokenData{ExpiresAt: now - 10},
			want:  false,
		},
		{
			name:  "valid with refresh token",
			token: TokenData{RefreshToken: "r", ExpiresAt: now + 3600},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.token.NeedsRefresh(); got != tt.want {
				t.Errorf("NeedsRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveLoadDeleteToken(t *testing.T) {
	home := setHome(t)

	token := &TokenData{
		Backend:       BackendMCP,
		AccessToken:   "access",
		TokenType:     "bearer",
		WorkspaceName: "ws",
		ClientID:      "client",
		RefreshToken:  "refresh",
		ExpiresAt:     12345,
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	// The token file must not be world-readable.
	info, err := os.Stat(filepath.Join(home, ".config", "gotion", TokenFileName))
	if err != nil {
		t.Fatalf("token file not written: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("token file permissions = %o, want %o", perm, 0o600)
	}

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error = %v", err)
	}
	if *got != *token {
		t.Errorf("LoadToken() = %+v, want %+v", *got, *token)
	}

	if err := DeleteToken(); err != nil {
		t.Fatalf("DeleteToken() error = %v", err)
	}
	if _, err := LoadToken(); err == nil {
		t.Error("LoadToken() after delete: error = nil, want error")
	}

	// Deleting a missing token file is not an error.
	if err := DeleteToken(); err != nil {
		t.Errorf("DeleteToken() on missing file error = %v, want nil", err)
	}
}

func TestLoadTokenInvalidJSON(t *testing.T) {
	home := setHome(t)
	dir := filepath.Join(home, ".config", "gotion")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, TokenFileName), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadToken(); err == nil {
		t.Error("LoadToken() error = nil, want error")
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	if err := (&Config{Token: "x"}).Validate(); err != nil {
		t.Errorf("Validate() with token error = %v, want nil", err)
	}
	if err := (&Config{}).Validate(); err == nil {
		t.Error("Validate() without token error = nil, want error")
	}
}

func TestConfigValidateOAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "valid",
			cfg:  Config{ClientID: "id", ClientSecret: "secret"},
		},
		{
			name:    "missing client id",
			cfg:     Config{ClientSecret: "secret"},
			wantErr: "api_client_id",
		},
		{
			name:    "missing client secret",
			cfg:     Config{ClientID: "id"},
			wantErr: "api_client_secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.ValidateOAuth()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateOAuth() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ValidateOAuth() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
