package notion

import (
	"strings"
	"testing"

	"github.com/longkey1/gotion/internal/gotion/config"
	"github.com/longkey1/gotion/internal/notion/api"
	"github.com/longkey1/gotion/internal/notion/mcp"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.Config
		check   func(t *testing.T, c Client)
		wantErr string
	}{
		{
			name: "api backend",
			cfg:  config.Config{Token: "t", Backend: config.BackendAPI},
			check: func(t *testing.T, c Client) {
				if _, ok := c.(*api.Client); !ok {
					t.Errorf("client type = %T, want *api.Client", c)
				}
			},
		},
		{
			name: "empty backend defaults to api",
			cfg:  config.Config{Token: "t"},
			check: func(t *testing.T, c Client) {
				if _, ok := c.(*api.Client); !ok {
					t.Errorf("client type = %T, want *api.Client", c)
				}
			},
		},
		{
			name: "mcp backend",
			cfg:  config.Config{Token: "t", Backend: config.BackendMCP},
			check: func(t *testing.T, c Client) {
				if _, ok := c.(*mcp.Client); !ok {
					t.Errorf("client type = %T, want *mcp.Client", c)
				}
			},
		},
		{
			name:    "missing token",
			cfg:     config.Config{Backend: config.BackendAPI},
			wantErr: "token is required",
		},
		{
			name:    "unknown backend",
			cfg:     config.Config{Token: "t", Backend: "grpc"},
			wantErr: "unknown backend: grpc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewClient(&tt.cfg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("NewClient() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			tt.check(t, got)
		})
	}
}
