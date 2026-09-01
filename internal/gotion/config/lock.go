package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	// LockFileName is the lock file used to serialize token refresh across processes
	LockFileName = "token.json.lock"

	// lockRetryInterval is the wait between lock acquisition attempts
	lockRetryInterval = 100 * time.Millisecond
)

// LockToken acquires an exclusive advisory lock (flock) on the token lock file,
// blocking until the lock is acquired or ctx is done. The returned function
// releases the lock. The lock file itself is left in place; deleting it on
// unlock would race with other processes waiting on the same file.
func LockToken(ctx context.Context) (func(), error) {
	if err := EnsureConfigDir(); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	lockPath := filepath.Join(configDir, LockFileName)

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return func() {
				_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
				_ = f.Close()
			}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			_ = f.Close()
			return nil, fmt.Errorf("failed to lock token file: %w", err)
		}

		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, fmt.Errorf("timed out waiting for token lock: %w", ctx.Err())
		case <-time.After(lockRetryInterval):
		}
	}
}
