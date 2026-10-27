package tts

import (
	"os"
	"testing"

	"github.com/devlikebear/break-reminder/internal/secrets"
)

// Tests never touch the real login keychain.
func TestMain(m *testing.M) {
	secrets.Default = secrets.NewMemory()
	os.Exit(m.Run())
}
