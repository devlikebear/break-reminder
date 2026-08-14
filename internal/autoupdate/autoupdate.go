package autoupdate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	cellarMarker = "/Cellar/break-reminder/"
	// Formula is the tap-qualified Homebrew formula updated by this package.
	Formula = "devlikebear/tap/break-reminder"
)

var (
	evalSymlinks = filepath.EvalSymlinks
	pathExists   = func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}
)

// HomebrewInstall describes the stable paths associated with a Homebrew install.
type HomebrewInstall struct {
	BrewPath    string
	BinaryPath  string
	MenuBarPath string
}

// RunCommand executes one Homebrew command and returns stdout.
type RunCommand func(context.Context, string, ...string) (string, error)

// Result describes whether a new formula version was installed.
type Result struct {
	Updated bool
}

// ExecuteCommand runs a Homebrew command with duplicate auto-update disabled.
func ExecuteCommand(ctx context.Context, path string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_ENV_HINTS=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return stdout.String(), fmt.Errorf("%w: %s", err, detail)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}

// CheckAndUpgrade refreshes Homebrew metadata and upgrades an outdated formula.
func CheckAndUpgrade(ctx context.Context, install HomebrewInstall, run RunCommand) (Result, error) {
	if _, err := run(ctx, install.BrewPath, "update"); err != nil {
		return Result{}, fmt.Errorf("brew update: %w", err)
	}
	outdated, err := run(ctx, install.BrewPath, "outdated", "--formula", "--quiet", Formula)
	if err != nil && !isOutdatedExit(outdated, err) {
		return Result{}, fmt.Errorf("brew outdated: %w", err)
	}
	if strings.TrimSpace(outdated) == "" {
		return Result{}, nil
	}
	if !listsFormula(outdated) {
		return Result{}, fmt.Errorf("brew outdated: unexpected output: %q", strings.TrimSpace(outdated))
	}
	if _, err := run(ctx, install.BrewPath, "upgrade", Formula); err != nil {
		return Result{}, fmt.Errorf("brew upgrade: %w", err)
	}
	return Result{Updated: true}, nil
}

func isOutdatedExit(output string, err error) bool {
	if !listsFormula(output) {
		return false
	}
	var exitCoder interface{ ExitCode() int }
	return errors.As(err, &exitCoder) && exitCoder.ExitCode() == 1
}

func listsFormula(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		switch strings.TrimSpace(line) {
		case Formula, "break-reminder":
			return true
		}
	}
	return false
}

// DetectHomebrewInstall identifies an executable installed in Homebrew's Cellar.
func DetectHomebrewInstall(executablePath string) (HomebrewInstall, bool) {
	resolved, err := evalSymlinks(executablePath)
	if err != nil {
		return HomebrewInstall{}, false
	}
	resolved = filepath.ToSlash(filepath.Clean(resolved))
	markerIndex := strings.Index(resolved, cellarMarker)
	if markerIndex <= 0 {
		return HomebrewInstall{}, false
	}

	prefix := resolved[:markerIndex]
	binDir := filepath.Join(prefix, "bin")
	brewPath := filepath.Join(binDir, "brew")
	binaryPath := filepath.Join(binDir, "break-reminder")
	menuBarPath := filepath.Join(binDir, "break-menubar")
	if !pathExists(brewPath) || !pathExists(binaryPath) {
		return HomebrewInstall{}, false
	}
	if !pathExists(menuBarPath) {
		menuBarPath = ""
	}
	return HomebrewInstall{
		BrewPath:    brewPath,
		BinaryPath:  binaryPath,
		MenuBarPath: menuBarPath,
	}, true
}
