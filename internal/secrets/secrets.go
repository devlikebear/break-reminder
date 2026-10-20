// Package secrets keeps API keys out of the config file.
//
// The direct-download build stores them in the login keychain (Security.framework
// through cgo) so the CLI and launchd `check` can read them without the app.
// The sandboxed app stores them with SecItem in Swift and hands them to the
// engine over RPC; those values live only in the in-memory override layer.
package secrets

import (
	"errors"
	"sync"

	"github.com/devlikebear/break-reminder/internal/i18n"
)

// Service is the keychain service name for every break-reminder secret.
const Service = "com.devlikebear.break-reminder"

// Known secret names.
const TTSAPIKey = "tts_api_key"

// ErrNotFound means the secret is not stored.
var ErrNotFound = errors.New(i18n.Text("secret not found"))

var errEmpty = errors.New(i18n.Text("secret value must not be empty"))

// Store reads and writes named secrets.
type Store interface {
	Get(name string) (string, error)
	Set(name, value string) error
	Delete(name string) error
}

// Memory is a process-local store.
type Memory struct {
	mu     sync.RWMutex
	values map[string]string
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory { return &Memory{values: map[string]string{}} }

func (m *Memory) Get(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.values[name]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (m *Memory) Set(name, value string) error {
	if value == "" {
		return errEmpty
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[name] = value
	return nil
}

func (m *Memory) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, name)
	return nil
}

// Layered reads Override first, then Base; writes go to Base.
type Layered struct {
	Override Store
	Base     Store
}

func (l Layered) Get(name string) (string, error) {
	if v, err := l.Override.Get(name); err == nil {
		return v, nil
	}
	return l.Base.Get(name)
}

func (l Layered) Set(name, value string) error { return l.Base.Set(name, value) }

func (l Layered) Delete(name string) error {
	_ = l.Override.Delete(name)
	return l.Base.Delete(name)
}

// Overrides holds values the app supplied over RPC (sandboxed build).
var Overrides = NewMemory()

// Default is the store the CLI and engine use.
var Default Store = Layered{Override: Overrides, Base: Keychain{Service: Service}}
