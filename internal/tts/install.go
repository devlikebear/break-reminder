package tts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devlikebear/break-reminder/internal/config"
)

const (
	defaultKittenVoice       = "Jasper"
	defaultKittenPackageSpec = "https://github.com/KittenML/KittenTTS/releases/download/0.8.1/kittentts-0.8.1-py3-none-any.whl"
)

// InstallOptions controls how KittenTTS is bootstrapped.
type InstallOptions struct {
	BootstrapPython string
	Model           string
	Voice           string
	Force           bool
}

// InstallResult captures the resulting runtime configuration after install.
type InstallResult struct {
	VenvDir    string
	PythonCmd  string
	Model      string
	Voice      string
	Installed  bool
	PackageRef string
}

// InstallKittenTTS bootstraps KittenTTS into an app-managed virtual environment.
func InstallKittenTTS(cfg config.Config, opts InstallOptions) (config.Config, InstallResult, error) {
	bootstrapPython := normalizePythonCommand(opts.BootstrapPython)
	venvDir := kittenInstallDir()
	venvPython := kittenVenvPythonPath(venvDir)

	result := InstallResult{
		VenvDir:    venvDir,
		PythonCmd:  venvPython,
		PackageRef: defaultKittenPackageSpec,
	}

	if _, err := exec.LookPath(bootstrapPython); err != nil {
		return cfg, result, fmt.Errorf("bootstrap python %q not found", bootstrapPython)
	}

	if err := ensureKittenVenv(bootstrapPython, venvDir, venvPython); err != nil {
		return cfg, result, err
	}

	needsInstall := opts.Force
	if !needsInstall {
		ok, err := kittenModuleInstalled(venvPython)
		if err != nil {
			return cfg, result, fmt.Errorf("check existing kittentts install: %w", err)
		}
		needsInstall = !ok
	}

	if needsInstall {
		if err := runInstallCommand(exec.Command(venvPython, "-m", "pip", "install", "--upgrade", "pip")); err != nil {
			return cfg, result, fmt.Errorf("upgrade pip: %w", err)
		}
		if err := runInstallCommand(exec.Command(venvPython, "-m", "pip", "install", "--upgrade", defaultKittenPackageSpec)); err != nil {
			return cfg, result, fmt.Errorf("install kittentts: %w", err)
		}
		result.Installed = true
	}

	ok, err := kittenModuleInstalled(venvPython)
	if err != nil {
		return cfg, result, fmt.Errorf("verify kittentts install: %w", err)
	}
	if !ok {
		return cfg, result, fmt.Errorf("kittentts installation verification failed")
	}

	updated := applyKittenConfig(cfg, opts.Voice, opts.Model, venvPython)
	result.Model = updated.TTSModel
	result.Voice = updated.Voice
	return updated, result, nil
}

func kittenInstallDir() string {
	return filepath.Join(config.ConfigDir(), "tts", "kittentts-venv")
}

func kittenVenvPythonPath(venvDir string) string {
	return filepath.Join(venvDir, "bin", "python")
}

func ensureKittenVenv(bootstrapPython, venvDir, venvPython string) error {
	if _, err := os.Stat(venvPython); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(venvDir), 0o755); err != nil {
		return err
	}

	cmd := exec.Command(bootstrapPython, "-m", "venv", venvDir)
	if err := runInstallCommand(cmd); err != nil {
		return fmt.Errorf("create virtualenv %s: %w", venvDir, err)
	}
	return nil
}

func runInstallCommand(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resolveKittenVoice(voice string) string {
	if canonical, ok := canonicalKittenVoice(voice); ok {
		return canonical
	}
	return defaultKittenVoice
}

func applyKittenConfig(cfg config.Config, preferredVoice, preferredModel, pythonCmd string) config.Config {
	updated := cfg
	updated.TTSEngine = engineKittenTTS
	updated.TTSPythonCmd = strings.TrimSpace(pythonCmd)

	model := strings.TrimSpace(preferredModel)
	if model == "" {
		model = strings.TrimSpace(updated.TTSModel)
	}
	updated.TTSModel = normalizeKittenModel(model)

	voice := strings.TrimSpace(preferredVoice)
	if voice == "" {
		voice = strings.TrimSpace(updated.Voice)
	}
	updated.Voice = resolveKittenVoice(voice)

	return updated
}
