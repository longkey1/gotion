package gotion

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// startCallbackServer starts a CallbackServer on a random port and
// returns it plus a channel delivering the Start result.
func startCallbackServer(t *testing.T, expectedState string) (*CallbackServer, <-chan error) {
	t.Helper()

	server, err := NewCallbackServer(0)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(context.Background(), expectedState)
	}()

	return server, errCh
}

// get performs an HTTP GET against the callback server.
func get(t *testing.T, server *CallbackServer, path string) *http.Response {
	t.Helper()

	url := fmt.Sprintf("http://127.0.0.1:%d%s", server.Port(), path)

	// The handler goroutine may not be serving yet; retry briefly.
	var resp *http.Response
	var err error
	for range 50 {
		resp, err = http.Get(url)
		if err == nil {
			return resp
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("GET %s failed: %v", url, err)
	return nil
}

func waitResult(t *testing.T, errCh <-chan error) error {
	t.Helper()

	select {
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Start() did not return within timeout")
		return nil
	}
}

func TestCallbackServerSuccess(t *testing.T) {
	t.Parallel()

	server, errCh := startCallbackServer(t, "expected-state")

	resp := get(t, server, "/callback?code=auth-code&state=expected-state")
	_ = resp.Body.Close()

	if err := waitResult(t, errCh); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got := server.Code(); got != "auth-code" {
		t.Errorf("Code() = %q, want %q", got, "auth-code")
	}
}

func TestCallbackServerEmptyExpectedStateSkipsCheck(t *testing.T) {
	t.Parallel()

	server, errCh := startCallbackServer(t, "")

	resp := get(t, server, "/callback?code=auth-code&state=whatever")
	_ = resp.Body.Close()

	if err := waitResult(t, errCh); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got := server.Code(); got != "auth-code" {
		t.Errorf("Code() = %q, want %q", got, "auth-code")
	}
}

func TestCallbackServerErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		expectedState string
		path          string
		wantErr       string
	}{
		{
			name:          "OAuth error parameter",
			expectedState: "s",
			path:          "/callback?error=access_denied",
			wantErr:       "OAuth error: access_denied",
		},
		{
			name:          "state mismatch",
			expectedState: "expected",
			path:          "/callback?code=c&state=wrong",
			wantErr:       "state mismatch",
		},
		{
			name:          "missing code",
			expectedState: "s",
			path:          "/callback?state=s",
			wantErr:       "no authorization code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, errCh := startCallbackServer(t, tt.expectedState)

			resp := get(t, server, tt.path)
			_ = resp.Body.Close()

			err := waitResult(t, errCh)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Start() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCallbackServerNonCallbackPathIs404(t *testing.T) {
	t.Parallel()

	server, errCh := startCallbackServer(t, "s")

	resp := get(t, server, "/other")
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET /other status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	// The server must still be waiting; complete the flow to shut down.
	resp = get(t, server, "/callback?code=c&state=s")
	_ = resp.Body.Close()
	if err := waitResult(t, errCh); err != nil {
		t.Errorf("Start() error = %v", err)
	}
}

func TestCallbackServerContextCancellation(t *testing.T) {
	t.Parallel()

	server, err := NewCallbackServer(0)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx, "s")
	}()

	cancel()

	if err := waitResult(t, errCh); err != context.Canceled {
		t.Errorf("Start() error = %v, want context.Canceled", err)
	}
}
