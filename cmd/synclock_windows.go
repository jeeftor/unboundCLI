//go:build windows

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

// acquireSyncLock acquires a non-blocking Windows file lock. The operating
// system releases it if this process exits unexpectedly.
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
	overlapped := new(windows.Overlapped)
	if err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlapped); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another sync is already running")
	}
	if err := f.Truncate(0); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("truncate lock file: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("seek lock file: %w", err)
	}
	if _, err := fmt.Fprintf(f, "%d\n", os.Getpid()); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("write PID to lock file: %w", err)
	}
	return func() { _ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, overlapped); _ = f.Close() }, nil
}

func getLockDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "AppData", "Local", "caddy-dns-sync")
	}
	return filepath.Join(os.TempDir(), "caddy-dns-sync")
}

const lockTimeout = 30 * time.Second

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
