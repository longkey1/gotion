package api

import (
	"net/url"
	"strings"
	"testing"
)

func TestOAuthClientGetAuthURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		state      string
		wantParams map[string]string
		noState    bool
	}{
		{
			name:  "with state",
			state: "abc123",
			wantParams: map[string]string{
				"client_id":     "my-client",
				"redirect_uri":  "http://localhost:8080/callback",
				"response_type": "code",
				"owner":         "user",
				"state":         "abc123",
			},
		},
		{
			name:    "without state",
			state:   "",
			noState: true,
			wantParams: map[string]string{
				"client_id":     "my-client",
				"redirect_uri":  "http://localhost:8080/callback",
				"response_type": "code",
				"owner":         "user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewOAuthClient(&OAuthConfig{
				ClientID:     "my-client",
				ClientSecret: "my-secret",
				RedirectURI:  "http://localhost:8080/callback",
			})

			got := c.GetAuthURL(tt.state)

			if !strings.HasPrefix(got, authURL+"?") {
				t.Fatalf("GetAuthURL() = %q, want prefix %q", got, authURL+"?")
			}

			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("GetAuthURL() returned unparsable URL: %v", err)
			}
			q := u.Query()
			for key, want := range tt.wantParams {
				if q.Get(key) != want {
					t.Errorf("param %s = %q, want %q", key, q.Get(key), want)
				}
			}
			if tt.noState {
				if _, ok := q["state"]; ok {
					t.Error("state param present, want absent")
				}
			}
		})
	}
}
