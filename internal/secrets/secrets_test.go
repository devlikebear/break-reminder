package secrets

import (
	"errors"
	"os"
	"testing"
)

func TestMemoryStore(t *testing.T) {
	m := NewMemory()
	if _, err := m.Get("k"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing = %v, want ErrNotFound", err)
	}
	if err := m.Set("k", "v1"); err != nil {
		t.Fatal(err)
	}
	if v, _ := m.Get("k"); v != "v1" {
		t.Fatalf("Get = %q", v)
	}
	if err := m.Delete("k"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get("k"); !errors.Is(err, ErrNotFound) {
		t.Fatal("Delete did not remove")
	}
}

func TestLayeredPrefersOverride(t *testing.T) {
	base := NewMemory()
	_ = base.Set("tts_api_key", "from-keychain")
	over := NewMemory()
	l := Layered{Override: over, Base: base}
	if v, _ := l.Get("tts_api_key"); v != "from-keychain" {
		t.Fatalf("fallback = %q", v)
	}
	_ = over.Set("tts_api_key", "from-app")
	if v, _ := l.Get("tts_api_key"); v != "from-app" {
		t.Fatalf("override = %q", v)
	}
	if err := l.Set("other", "x"); err != nil {
		t.Fatal(err)
	}
	if v, _ := base.Get("other"); v != "x" {
		t.Fatal("Set should persist to the base store")
	}
}

func TestEmptyValueIsRejected(t *testing.T) {
	if err := NewMemory().Set("k", ""); err == nil {
		t.Fatal("empty secret accepted")
	}
}

// Touches the real login keychain, so it only runs when asked explicitly.
func TestKeychainRoundTrip(t *testing.T) {
	if os.Getenv("BREAK_REMINDER_KEYCHAIN_TEST") != "1" {
		t.Skip("set BREAK_REMINDER_KEYCHAIN_TEST=1 to exercise the login keychain")
	}
	k := Keychain{Service: Service + ".test"}
	defer k.Delete("roundtrip")
	if err := k.Set("roundtrip", "one"); err != nil {
		t.Fatal(err)
	}
	if err := k.Set("roundtrip", "two"); err != nil {
		t.Fatal(err)
	}
	if v, err := k.Get("roundtrip"); err != nil || v != "two" {
		t.Fatalf("Get = %q, %v", v, err)
	}
	if err := k.Delete("roundtrip"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Get("roundtrip"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete = %v", err)
	}
}
