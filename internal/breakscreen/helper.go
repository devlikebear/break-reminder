package breakscreen

import (
	"os"
	"os/exec"
	"path/filepath"
)

// FindHelper searches for a named helper binary in common locations.
func FindHelper(name string) string {
	// 1. Next to the main binary
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// 2. Common development / install paths
	candidates := []string{
		"bin/" + name,
		filepath.Join(os.Getenv("HOME"), ".local", "bin", name),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	// 3. In PATH
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	return ""
}
