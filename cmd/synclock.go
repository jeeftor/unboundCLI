package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// acquireSyncLock acquires an exclusive file lock to prevent concurrent syncs.
// Returns a cleanup function that releases the lock.
func acquireSyncLock() (func(), error) {
	lockDir := getLockDir()
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating lock directory: %w", err)
	}
	lockPath := filepath.Join(lockDir, "sync.lock")

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}

	// Non-blocking lock attempt.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		// Read the PID from the existing lock file for a helpful message.
		data, _ := os.ReadFile(lockPath)
		pid := string(data)
		f.Close()
		if pid != "" {
			return nil, fmt.Errorf("another sync is already running (PID %s) — remove %s if stale", pid, lockPath)
		}
		return nil, fmt.Errorf("another sync is already running — remove %s if stale", lockPath)
	}

	// Write our PID.
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())

	cleanup := func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
	return cleanup, nil
}

// getLockDir returns the directory for the sync lock file.
func getLockDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "caddy-dns-sync")
	}
	return filepath.Join(os.TempDir(), "caddy-dns-sync")
}

// lockTimeout is how long to wait when acquiring a lock with retry.
const lockTimeout = 30 * time.Second

// acquireSyncLockWithWait tries to acquire the sync lock, waiting up to lockTimeout.
func acquireSyncLockWithWait() (func(), error) {
	deadline := time.Now().Add(lockTimeout)
	for {
		cleanup, err := acquireSyncLock()
		if err == nil {
			return cleanup, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// suppress unused warning for strconv import (used in future enhancements)
var _ = strconv.Itoa
