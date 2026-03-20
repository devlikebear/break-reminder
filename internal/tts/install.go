package tts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/devlikebear/break-reminder/internal/config"
)

const (
	defaultKittenVoice           = "Jasper"
	defaultKittenPackageSpec     = "https://github.com/KittenML/KittenTTS/releases/download/0.8.1/kittentts-0.8.1-py3-none-any.whl"
	defaultSupertonicVoice       = "M1"
	defaultSupertonicPackageSpec = "supertonic==1.1.2"
)

// InstallOptions controls how an optional Python-backed TTS engine is bootstrapped.
type InstallOptions struct {
	BootstrapPython         string
	BootstrapPythonExplicit bool
	Model                   string
	Voice                   string
	Force                   bool
}

// InstallResult captures the resulting runtime configuration after install.
type InstallResult struct {
	VenvDir         string
	PythonCmd       string
	BootstrapPython string
	Model           string
	Voice           string
	Installed       bool
	PackageRef      string
}

// UninstallResult captures the outcome of removing a managed TTS environment.
type UninstallResult struct {
	VenvDir       string
	Removed       bool
	ConfigUpdated bool
}

// InstallKittenTTS bootstraps KittenTTS into an app-managed virtual environment.
func InstallKittenTTS(cfg config.Config, opts InstallOptions) (config.Config, InstallResult, error) {
	venvDir := kittenInstallDir()
	venvPython := kittenVenvPythonPath(venvDir)

	result := InstallResult{
		VenvDir:    venvDir,
		PythonCmd:  venvPython,
		PackageRef: defaultKittenPackageSpec,
	}

	bootstrapPython, err := resolveBootstrapPython(opts.BootstrapPython, opts.BootstrapPythonExplicit)
	if err != nil {
		return cfg, result, err
	}
	result.BootstrapPython = bootstrapPython.command

	if err := ensureManagedVenv(bootstrapPython, venvDir, venvPython); err != nil {
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

// InstallSupertonic bootstraps Supertonic into an app-managed virtual environment.
func InstallSupertonic(cfg config.Config, opts InstallOptions) (config.Config, InstallResult, error) {
	venvDir := supertonicInstallDir()
	venvPython := supertonicVenvPythonPath(venvDir)

	result := InstallResult{
		VenvDir:    venvDir,
		PythonCmd:  venvPython,
		PackageRef: defaultSupertonicPackageSpec,
	}

	bootstrapPython, err := resolveBootstrapPython(opts.BootstrapPython, opts.BootstrapPythonExplicit)
	if err != nil {
		return cfg, result, err
	}
	result.BootstrapPython = bootstrapPython.command

	if err := ensureManagedVenv(bootstrapPython, venvDir, venvPython); err != nil {
		return cfg, result, err
	}

	needsInstall := opts.Force
	if !needsInstall {
		ok, err := supertonicModuleInstalled(venvPython)
		if err != nil {
			return cfg, result, fmt.Errorf("check existing supertonic install: %w", err)
		}
		needsInstall = !ok
	}

	if needsInstall {
		if err := runInstallCommand(exec.Command(venvPython, "-m", "pip", "install", "--upgrade", "pip")); err != nil {
			return cfg, result, fmt.Errorf("upgrade pip: %w", err)
		}
		if err := runInstallCommand(exec.Command(venvPython, "-m", "pip", "install", "--upgrade", defaultSupertonicPackageSpec)); err != nil {
			return cfg, result, fmt.Errorf("install supertonic: %w", err)
		}
		result.Installed = true
	}

	ok, err := supertonicModuleInstalled(venvPython)
	if err != nil {
		return cfg, result, fmt.Errorf("verify supertonic install: %w", err)
	}
	if !ok {
		return cfg, result, fmt.Errorf("supertonic installation verification failed")
	}

	updated := applySupertonicConfig(cfg, opts.Voice, opts.Model, venvPython)
	result.Model = updated.TTSModel
	result.Voice = updated.Voice
	return updated, result, nil
}

// UninstallKittenTTS removes the managed KittenTTS environment and restores config when active.
func UninstallKittenTTS(cfg config.Config) (config.Config, UninstallResult, error) {
	venvDir := kittenInstallDir()
	result := UninstallResult{VenvDir: venvDir}

	if _, err := os.Stat(venvDir); err == nil {
		if err := os.RemoveAll(venvDir); err != nil {
			return cfg, result, fmt.Errorf("remove managed KittenTTS environment %s: %w", venvDir, err)
		}
		result.Removed = true
	} else if !os.IsNotExist(err) {
		return cfg, result, err
	}

	updated := cfg
	if shouldResetAfterKittenUninstall(cfg) {
		updated = applySayConfig(cfg)
		result.ConfigUpdated = true
	}

	return updated, result, nil
}

// UninstallSupertonic removes the managed Supertonic environment and restores config when active.
func UninstallSupertonic(cfg config.Config) (config.Config, UninstallResult, error) {
	venvDir := supertonicInstallDir()
	result := UninstallResult{VenvDir: venvDir}

	if _, err := os.Stat(venvDir); err == nil {
		if err := os.RemoveAll(venvDir); err != nil {
			return cfg, result, fmt.Errorf("remove managed Supertonic environment %s: %w", venvDir, err)
		}
		result.Removed = true
	} else if !os.IsNotExist(err) {
		return cfg, result, err
	}

	updated := cfg
	if shouldResetAfterSupertonicUninstall(cfg) {
		updated = applySayConfig(cfg)
		result.ConfigUpdated = true
	}

	return updated, result, nil
}

func kittenInstallDir() string {
	return filepath.Join(config.ConfigDir(), "tts", "kittentts-venv")
}

func kittenVenvPythonPath(venvDir string) string {
	return filepath.Join(venvDir, "bin", "python")
}

func supertonicInstallDir() string {
	return filepath.Join(config.ConfigDir(), "tts", "supertonic-venv")
}

func supertonicVenvPythonPath(venvDir string) string {
	return filepath.Join(venvDir, "bin", "python")
}

func ensureManagedVenv(bootstrapPython pythonCandidate, venvDir, venvPython string) error {
	if _, err := os.Stat(venvPython); err == nil {
		version, versionErr := pythonVersionOf(venvPython)
		if versionErr == nil && supportsManagedPythonVersion(version) {
			return nil
		}
		if err := os.RemoveAll(venvDir); err != nil {
			return fmt.Errorf("reset incompatible virtualenv %s: %w", venvDir, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(venvDir), 0o755); err != nil {
		return err
	}

	cmd := exec.Command(bootstrapPython.command, "-m", "venv", venvDir)
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

func resolveSupertonicVoice(voice string) string {
	if canonical, ok := canonicalSupertonicVoice(voice); ok {
		return canonical
	}
	return defaultSupertonicVoice
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

func applySupertonicConfig(cfg config.Config, preferredVoice, preferredModel, pythonCmd string) config.Config {
	updated := cfg
	updated.TTSEngine = engineSupertonic
	updated.TTSPythonCmd = strings.TrimSpace(pythonCmd)

	model := strings.TrimSpace(preferredModel)
	if model == "" {
		currentModel := strings.TrimSpace(updated.TTSModel)
		if normalizeEngine(cfg.TTSEngine) == engineSupertonic || strings.HasPrefix(strings.ToLower(currentModel), "supertone/") {
			model = currentModel
		}
	}
	updated.TTSModel = normalizeSupertonicModel(model)

	voice := strings.TrimSpace(preferredVoice)
	if voice == "" {
		voice = strings.TrimSpace(updated.Voice)
	}
	updated.Voice = resolveSupertonicVoice(voice)

	return updated
}

func applySayConfig(cfg config.Config) config.Config {
	defaults := config.Default()
	updated := cfg
	updated.TTSEngine = engineSay
	updated.TTSModel = defaults.TTSModel
	updated.TTSPythonCmd = defaults.TTSPythonCmd
	updated.Voice = defaults.Voice
	return updated
}

func shouldResetAfterKittenUninstall(cfg config.Config) bool {
	return shouldResetAfterManagedUninstall(cfg, engineKittenTTS, kittenVenvPythonPath(kittenInstallDir()))
}

func shouldResetAfterSupertonicUninstall(cfg config.Config) bool {
	return shouldResetAfterManagedUninstall(cfg, engineSupertonic, supertonicVenvPythonPath(supertonicInstallDir()))
}

func shouldResetAfterManagedUninstall(cfg config.Config, engine, managedPython string) bool {
	if normalizeEngine(cfg.TTSEngine) == engine {
		return true
	}
	currentPython := strings.TrimSpace(cfg.TTSPythonCmd)
	if currentPython == "" {
		return false
	}
	return filepath.Clean(currentPython) == filepath.Clean(managedPython)
}

type pythonVersion struct {
	major int
	minor int
}

func (v pythonVersion) String() string {
	if v.major == 0 {
		return "unknown"
	}
	return fmt.Sprintf("%d.%d", v.major, v.minor)
}

type pythonCandidate struct {
	command string
	version pythonVersion
}

func supportsManagedPythonVersion(version pythonVersion) bool {
	return version.major == 3 && version.minor >= 8 && version.minor < 13
}

func supportsKittenPythonVersion(version pythonVersion) bool {
	return supportsManagedPythonVersion(version)
}

func selectCompatiblePython(candidates []pythonCandidate) (pythonCandidate, bool) {
	for _, candidate := range candidates {
		if supportsManagedPythonVersion(candidate.version) {
			return candidate, true
		}
	}
	return pythonCandidate{}, false
}

func resolveBootstrapPython(requested string, explicit bool) (pythonCandidate, error) {
	if explicit {
		candidate, err := inspectPythonCandidate(requested)
		if err != nil {
			return pythonCandidate{}, fmt.Errorf("bootstrap python %q not found", requested)
		}
		if !supportsManagedPythonVersion(candidate.version) {
			return pythonCandidate{}, fmt.Errorf(
				"bootstrap python %q uses Python %s; managed Python TTS engines currently need Python 3.8-3.12-compatible dependencies",
				candidate.command,
				candidate.version,
			)
		}
		return candidate, nil
	}

	var candidates []pythonCandidate
	for _, command := range []string{"python3.12", "python3.11", "python3.10", "python3.9", "python3.8", "python3"} {
		candidate, err := inspectPythonCandidate(command)
		if err == nil {
			candidates = append(candidates, candidate)
		}
	}

	if candidate, ok := selectCompatiblePython(candidates); ok {
		return candidate, nil
	}
	if len(candidates) == 0 {
		return pythonCandidate{}, fmt.Errorf("no Python interpreter found; install Python 3.8-3.12 or pass --bootstrap-python")
	}

	var found []string
	for _, candidate := range candidates {
		found = append(found, fmt.Sprintf("%s (Python %s)", candidate.command, candidate.version))
	}
	return pythonCandidate{}, fmt.Errorf(
		"no compatible Python interpreter found for managed Python TTS engines; found %s, need Python 3.8-3.12",
		strings.Join(found, ", "),
	)
}

func inspectPythonCandidate(command string) (pythonCandidate, error) {
	resolved, err := exec.LookPath(strings.TrimSpace(command))
	if err != nil {
		return pythonCandidate{}, err
	}

	version, err := pythonVersionOf(resolved)
	if err != nil {
		return pythonCandidate{}, err
	}

	return pythonCandidate{
		command: resolved,
		version: version,
	}, nil
}

func pythonVersionOf(command string) (pythonVersion, error) {
	out, err := exec.Command(
		command,
		"-c",
		"import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')",
	).Output()
	if err != nil {
		return pythonVersion{}, err
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ".")
	if len(parts) != 2 {
		return pythonVersion{}, fmt.Errorf("unexpected python version output %q", strings.TrimSpace(string(out)))
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return pythonVersion{}, err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return pythonVersion{}, err
	}
	return pythonVersion{major: major, minor: minor}, nil
}
