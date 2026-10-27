// Package enginelock lets the long-running engine announce itself so that the
// launchd `check` job and the standalone time-tools worker stand aside while
// it owns the timer state.
package enginelock

import (
	"github.com/devlikebear/break-reminder/internal/i18n"

	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// ErrHeld is returned by Acquire when another engine holds the lock.
var ErrHeld = errors.New(i18n.Text("Another break-reminder engine is already running"))

// Lock is an acquired engine lock.
type Lock struct{ f *os.File }

// Acquire takes the lock without blocking.
func Acquire(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrHeld
		}
		return nil, err
	}
	return &Lock{f: f}, nil
}

// Release drops the lock. It is safe to call on a nil Lock.
func (l *Lock) Release() {
	if l == nil || l.f == nil {
		return
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	_ = l.f.Close()
	l.f = nil
}

// Held reports whether some process currently holds the lock at path.
func Held(path string) bool {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return errors.Is(err, syscall.EWOULDBLOCK)
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false
}
