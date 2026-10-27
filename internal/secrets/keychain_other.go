//go:build !darwin || !cgo

package secrets

import (
	"errors"

	"github.com/devlikebear/break-reminder/internal/i18n"
)

// Keychain is unavailable on this platform; every lookup misses.
type Keychain struct{ Service string }

var errNoKeychain = errors.New(i18n.Text("Keychain is unavailable on this platform"))

func (Keychain) Get(string) (string, error) { return "", ErrNotFound }
func (Keychain) Set(string, string) error   { return errNoKeychain }
func (Keychain) Delete(string) error        { return nil }
