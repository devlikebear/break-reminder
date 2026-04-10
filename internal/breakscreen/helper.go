package breakscreen

import (
	"os"
	"path/filepath"
)

var executablePath = os.Executable
var userHomeDir = os.UserHomeDir

func trustedHelperCandidates(name string) []string {
	var candidates []string

	if exe, err := executablePath(); err == nil && exe != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), name))
	}

	if home, err := userHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", name))
	}

	return candidates
}

// FindHelper searches for a named helper binary in trusted install locations.
func FindHelper(name string) string {
	for _, candidate := range trustedHelperCandidates(name) {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		if abs, err := filepath.Abs(candidate); err == nil {
			return abs
		}
		return candidate
	}

	return ""
}
