package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLockToken(t *testing.T) {
	t.Run("acquire and release", func(t *testing.T) {
		setHome(t)

		unlock, err := LockToken(context.Background())
		if err != nil {
			t.Fatalf("LockToken() error = %v", err)
		}
		unlock()

		// Reacquirable after release
		unlock, err = LockToken(context.Background())
		if err != nil {
			t.Fatalf("LockToken() after unlock error = %v", err)
		}
		unlock()
	})

	t.Run("blocks while held", func(t *testing.T) {
		setHome(t)

		unlock, err := LockToken(context.Background())
		if err != nil {
			t.Fatalf("LockToken() error = %v", err)
		}
		defer unlock()

		// flock locks are per open file description, so a second LockToken
		// in the same process conflicts just like another process would
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if _, err := LockToken(ctx); err == nil {
			t.Fatal("LockToken() succeeded while lock was held, want timeout error")
		}
	})

	t.Run("acquired after holder releases", func(t *testing.T) {
		setHome(t)

		unlock, err := LockToken(context.Background())
		if err != nil {
			t.Fatalf("LockToken() error = %v", err)
		}

		go func() {
			time.Sleep(200 * time.Millisecond)
			unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		unlock2, err := LockToken(ctx)
		if err != nil {
			t.Fatalf("LockToken() waiting for release error = %v", err)
		}
		unlock2()
	})
}

func TestSaveTokenLeavesNoTempFiles(t *testing.T) {
	home := setHome(t)

	if err := SaveToken(&TokenData{Backend: BackendMCP, AccessToken: "tok"}); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(home, ".config", "gotion"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}
