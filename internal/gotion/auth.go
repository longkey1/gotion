package gotion

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

// CallbackServer handles the OAuth callback
type CallbackServer struct {
	port     int
	listener net.Listener
	code     string
	state    string
	err      error
	done     chan struct{}
}

// NewCallbackServer creates a new callback server
func NewCallbackServer(port int) (*CallbackServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	return &CallbackServer{
		port:     port,
		listener: listener,
		done:     make(chan struct{}),
	}, nil
}

// Port returns the actual port the server is listening on
func (s *CallbackServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// Start starts the callback server and waits for the callback
func (s *CallbackServer) Start(ctx context.Context, expectedState string) error {
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			query := r.URL.Query()

			// Check for error
			if errCode := query.Get("error"); errCode != "" {
				s.err = fmt.Errorf("OAuth error: %s", errCode)
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<html><body><h1>Authentication Failed</h1><p>%s</p><p>You can close this window.</p></body></html>`, errCode)
				close(s.done)
				return
			}

			// Verify state
			state := query.Get("state")
			if expectedState != "" && state != expectedState {
				s.err = fmt.Errorf("state mismatch")
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><body><h1>Authentication Failed</h1><p>State mismatch</p><p>You can close this window.</p></body></html>`)
				close(s.done)
				return
			}

			// Get authorization code
			code := query.Get("code")
			if code == "" {
				s.err = fmt.Errorf("no authorization code received")
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><body><h1>Authentication Failed</h1><p>No authorization code received</p><p>You can close this window.</p></body></html>`)
				close(s.done)
				return
			}

			s.code = code
			s.state = state
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h1>Authentication Successful!</h1><p>You can close this window and return to the terminal.</p></body></html>`)
			close(s.done)
		}),
	}

	go func() {
		_ = server.Serve(s.listener)
	}()

	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return ctx.Err()
	case <-s.done:
		_ = server.Shutdown(context.Background())
		return s.err
	}
}

// Code returns the authorization code received
func (s *CallbackServer) Code() string {
	return s.code
}

// Close closes the callback server
func (s *CallbackServer) Close() error {
	return s.listener.Close()
}
