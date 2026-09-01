package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// newMetadataServer starts a server that serves RFC 9728 / RFC 8414
// discovery metadata plus registration and token endpoints, all pointing
// back at itself.
func newMetadataServer(t *testing.T, tokenHandler http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ProtectedResourceMetadata{
			Resource:             srv.URL,
			AuthorizationServers: []string{srv.URL},
		})
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(AuthServerMetadata{
			Issuer:                srv.URL,
			AuthorizationEndpoint: srv.URL + "/authorize",
			TokenEndpoint:         srv.URL + "/token",
			RegistrationEndpoint:  srv.URL + "/register",
		})
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		var req ClientRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ClientRegistrationResponse{
			ClientID:     "registered-client",
			RedirectURIs: req.RedirectURIs,
			ClientName:   req.ClientName,
		})
	})
	if tokenHandler != nil {
		mux.HandleFunc("/token", tokenHandler)
	}

	return srv
}

// newTestOAuthClient builds an OAuthClient pointed at the test server.
func newTestOAuthClient(srv *httptest.Server) *OAuthClient {
	return &OAuthClient{
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		mcpServerURL: srv.URL,
		callbackURL:  "http://127.0.0.1:9998/callback",
	}
}

func TestNewOAuthClientDefaultCallbackURL(t *testing.T) {
	t.Parallel()

	if got := NewOAuthClient("").GetCallbackURL(); got != defaultCallbackURL {
		t.Errorf("GetCallbackURL() = %q, want %q", got, defaultCallbackURL)
	}
	if got := NewOAuthClient("http://localhost:1234/cb").GetCallbackURL(); got != "http://localhost:1234/cb" {
		t.Errorf("GetCallbackURL() = %q, want %q", got, "http://localhost:1234/cb")
	}
}

func TestGeneratePKCE(t *testing.T) {
	t.Parallel()

	c := NewOAuthClient("")
	if err := c.GeneratePKCE(); err != nil {
		t.Fatalf("GeneratePKCE() error = %v", err)
	}
	if c.pkce == nil {
		t.Fatal("pkce = nil after GeneratePKCE()")
	}
	if c.pkce.CodeVerifier == "" {
		t.Fatal("CodeVerifier is empty")
	}

	// code_challenge must be BASE64URL(SHA256(code_verifier)) per RFC 7636.
	hash := sha256.Sum256([]byte(c.pkce.CodeVerifier))
	want := base64.RawURLEncoding.EncodeToString(hash[:])
	if c.pkce.CodeChallenge != want {
		t.Errorf("CodeChallenge = %q, want %q", c.pkce.CodeChallenge, want)
	}
}

func TestDiscoverEndpoints(t *testing.T) {
	t.Parallel()

	srv := newMetadataServer(t, nil)
	c := newTestOAuthClient(srv)

	if err := c.DiscoverEndpoints(context.Background()); err != nil {
		t.Fatalf("DiscoverEndpoints() error = %v", err)
	}
	if c.authServer.AuthorizationEndpoint != srv.URL+"/authorize" {
		t.Errorf("AuthorizationEndpoint = %q, want %q", c.authServer.AuthorizationEndpoint, srv.URL+"/authorize")
	}
	if c.authServer.TokenEndpoint != srv.URL+"/token" {
		t.Errorf("TokenEndpoint = %q, want %q", c.authServer.TokenEndpoint, srv.URL+"/token")
	}
}

func TestDiscoverEndpointsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{
			name: "protected resource not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantErr: "failed to fetch protected resource metadata",
		},
		{
			name: "no authorization servers",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(ProtectedResourceMetadata{Resource: "x"})
			},
			wantErr: "no authorization servers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := newTestOAuthClient(srv)
			err := c.DiscoverEndpoints(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("DiscoverEndpoints() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestDiscoverEndpointsMissingEndpoints(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ProtectedResourceMetadata{AuthorizationServers: []string{srv.URL}})
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(AuthServerMetadata{Issuer: srv.URL})
	})

	c := newTestOAuthClient(srv)
	err := c.DiscoverEndpoints(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing required endpoints") {
		t.Errorf("DiscoverEndpoints() error = %v, want missing endpoints error", err)
	}
}

func TestRegisterClient(t *testing.T) {
	t.Parallel()

	srv := newMetadataServer(t, nil)
	c := newTestOAuthClient(srv)
	ctx := context.Background()

	// Registration before discovery must fail.
	if err := c.RegisterClient(ctx); err == nil {
		t.Error("RegisterClient() before discovery: error = nil, want error")
	}

	if err := c.DiscoverEndpoints(ctx); err != nil {
		t.Fatalf("DiscoverEndpoints() error = %v", err)
	}
	if err := c.RegisterClient(ctx); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	if got := c.GetClientID(); got != "registered-client" {
		t.Errorf("GetClientID() = %q, want %q", got, "registered-client")
	}
}

func TestGetClientIDBeforeRegistration(t *testing.T) {
	t.Parallel()

	if got := NewOAuthClient("").GetClientID(); got != "" {
		t.Errorf("GetClientID() = %q, want empty string", got)
	}
}

func TestGetAuthURL(t *testing.T) {
	t.Parallel()

	c := &OAuthClient{
		callbackURL: "http://127.0.0.1:9998/callback",
		authServer:  &AuthServerMetadata{AuthorizationEndpoint: "https://auth.example.com/authorize"},
		clientReg:   &ClientRegistrationResponse{ClientID: "my-client"},
		pkce:        &PKCEPair{CodeVerifier: "v", CodeChallenge: "challenge"},
	}

	got, err := c.GetAuthURL("state-1")
	if err != nil {
		t.Fatalf("GetAuthURL() error = %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("GetAuthURL() returned unparsable URL: %v", err)
	}
	q := u.Query()
	for key, want := range map[string]string{
		"client_id":             "my-client",
		"redirect_uri":          "http://127.0.0.1:9998/callback",
		"response_type":         "code",
		"code_challenge":        "challenge",
		"code_challenge_method": "S256",
		"state":                 "state-1",
	} {
		if q.Get(key) != want {
			t.Errorf("param %s = %q, want %q", key, q.Get(key), want)
		}
	}
}

func TestGetAuthURLPreconditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		client  *OAuthClient
		wantErr string
	}{
		{
			name:    "auth server not discovered",
			client:  &OAuthClient{},
			wantErr: "not discovered",
		},
		{
			name: "client not registered",
			client: &OAuthClient{
				authServer: &AuthServerMetadata{AuthorizationEndpoint: "x"},
			},
			wantErr: "not registered",
		},
		{
			name: "PKCE not generated",
			client: &OAuthClient{
				authServer: &AuthServerMetadata{AuthorizationEndpoint: "x"},
				clientReg:  &ClientRegistrationResponse{ClientID: "c"},
			},
			wantErr: "PKCE not generated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.client.GetAuthURL("s")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("GetAuthURL() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestExchangeCode(t *testing.T) {
	t.Parallel()

	var gotForm url.Values
	srv := newMetadataServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotForm = r.PostForm
		_ = json.NewEncoder(w).Encode(OAuthToken{
			AccessToken:  "access-token",
			TokenType:    "bearer",
			RefreshToken: "refresh-token",
			ExpiresIn:    3600,
		})
	})

	c := newTestOAuthClient(srv)
	ctx := context.Background()
	if err := c.DiscoverEndpoints(ctx); err != nil {
		t.Fatalf("DiscoverEndpoints() error = %v", err)
	}
	if err := c.RegisterClient(ctx); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	if err := c.GeneratePKCE(); err != nil {
		t.Fatalf("GeneratePKCE() error = %v", err)
	}

	before := time.Now().Unix()
	token, err := c.ExchangeCode(ctx, "auth-code")
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}

	if token.AccessToken != "access-token" {
		t.Errorf("AccessToken = %q, want %q", token.AccessToken, "access-token")
	}
	if token.RefreshToken != "refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", token.RefreshToken, "refresh-token")
	}
	if token.ExpiresAt < before+3600 {
		t.Errorf("ExpiresAt = %d, want >= %d (now + expires_in)", token.ExpiresAt, before+3600)
	}

	for key, want := range map[string]string{
		"grant_type":    "authorization_code",
		"code":          "auth-code",
		"client_id":     "registered-client",
		"code_verifier": c.pkce.CodeVerifier,
	} {
		if gotForm.Get(key) != want {
			t.Errorf("form %s = %q, want %q", key, gotForm.Get(key), want)
		}
	}
}

func TestExchangeCodeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{
			name: "HTTP error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "invalid_grant", http.StatusBadRequest)
			},
			wantErr: "HTTP 400",
		},
		{
			name: "missing access token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(OAuthToken{})
			},
			wantErr: "no access_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := newMetadataServer(t, tt.handler)
			c := newTestOAuthClient(srv)
			ctx := context.Background()
			if err := c.DiscoverEndpoints(ctx); err != nil {
				t.Fatalf("DiscoverEndpoints() error = %v", err)
			}
			if err := c.RegisterClient(ctx); err != nil {
				t.Fatalf("RegisterClient() error = %v", err)
			}
			if err := c.GeneratePKCE(); err != nil {
				t.Fatalf("GeneratePKCE() error = %v", err)
			}

			_, err := c.ExchangeCode(ctx, "auth-code")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ExchangeCode() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseTokenEndpointError(t *testing.T) {
	t.Parallel()

	t.Run("RFC 6749 error object", func(t *testing.T) {
		t.Parallel()

		err := parseTokenEndpointError(400, []byte(`{"error":"invalid_grant","error_description":"refresh token expired"}`))

		oauthErr, ok := errors.AsType[*OAuthError](err)
		if !ok {
			t.Fatalf("parseTokenEndpointError() = %v, want *OAuthError", err)
		}
		if oauthErr.StatusCode != 400 {
			t.Errorf("StatusCode = %d, want 400", oauthErr.StatusCode)
		}
		if !oauthErr.IsInvalidGrant() {
			t.Errorf("IsInvalidGrant() = false, want true (code = %q)", oauthErr.Code)
		}
		if !strings.Contains(oauthErr.Error(), "refresh token expired") {
			t.Errorf("Error() = %q, want it to contain the description", oauthErr.Error())
		}
	})

	t.Run("other error code is not invalid_grant", func(t *testing.T) {
		t.Parallel()

		err := parseTokenEndpointError(400, []byte(`{"error":"invalid_client"}`))

		oauthErr, ok := errors.AsType[*OAuthError](err)
		if !ok {
			t.Fatalf("parseTokenEndpointError() = %v, want *OAuthError", err)
		}
		if oauthErr.IsInvalidGrant() {
			t.Error("IsInvalidGrant() = true, want false")
		}
	})

	t.Run("non-JSON body falls back to generic error", func(t *testing.T) {
		t.Parallel()

		err := parseTokenEndpointError(502, []byte("Bad Gateway"))

		if _, ok := errors.AsType[*OAuthError](err); ok {
			t.Fatalf("parseTokenEndpointError() = *OAuthError, want generic error")
		}
		if !strings.Contains(err.Error(), "HTTP 502") || !strings.Contains(err.Error(), "Bad Gateway") {
			t.Errorf("Error() = %q, want status and body included", err.Error())
		}
	})
}
