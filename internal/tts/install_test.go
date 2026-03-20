package tts

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/devlikebear/break-reminder/internal/config"
)

func TestResolveKittenVoice(t *testing.T) {
	if got := resolveKittenVoice(" bella "); got != "Bella" {
		t.Fatalf("resolveKittenVoice() = %q, want Bella", got)
	}
	if got := resolveKittenVoice("Yuna"); got != defaultKittenVoice {
		t.Fatalf("resolveKittenVoice() = %q, want %q", got, defaultKittenVoice)
	}
}

func TestResolveSupertonicVoice(t *testing.T) {
	if got := resolveSupertonicVoice(" m1 "); got != "M1" {
		t.Fatalf("resolveSupertonicVoice() = %q, want M1", got)
	}
	if got := resolveSupertonicVoice("Yuna"); got != defaultSupertonicVoice {
		t.Fatalf("resolveSupertonicVoice() = %q, want %q", got, defaultSupertonicVoice)
	}
}

func TestKittenVenvPythonPath(t *testing.T) {
	got := kittenVenvPythonPath("/tmp/break-reminder/kittentts-venv")
	want := filepath.Join("/tmp/break-reminder/kittentts-venv", "bin", "python")
	if got != want {
		t.Fatalf("kittenVenvPythonPath() = %q, want %q", got, want)
	}
}

func TestKittenInstallDir(t *testing.T) {
	got := kittenInstallDir()
	if !strings.HasSuffix(got, filepath.Join("break-reminder", "tts", "kittentts-venv")) {
		t.Fatalf("kittenInstallDir() = %q, want suffix %q", got, filepath.Join("break-reminder", "tts", "kittentts-venv"))
	}
}

func TestSupertonicInstallDir(t *testing.T) {
	got := supertonicInstallDir()
	if !strings.HasSuffix(got, filepath.Join("break-reminder", "tts", "supertonic-venv")) {
		t.Fatalf("supertonicInstallDir() = %q, want suffix %q", got, filepath.Join("break-reminder", "tts", "supertonic-venv"))
	}
}

func TestApplyKittenConfigUsesSafeDefaults(t *testing.T) {
	cfg := config.Default()
	cfg.Voice = "Yuna"
	cfg.TTSModel = ""

	updated := applyKittenConfig(cfg, "", "", "/tmp/kitten/bin/python")

	if updated.TTSEngine != engineKittenTTS {
		t.Fatalf("TTSEngine = %q, want %q", updated.TTSEngine, engineKittenTTS)
	}
	if updated.TTSPythonCmd != "/tmp/kitten/bin/python" {
		t.Fatalf("TTSPythonCmd = %q, want /tmp/kitten/bin/python", updated.TTSPythonCmd)
	}
	if updated.TTSModel != defaultKittenModel {
		t.Fatalf("TTSModel = %q, want %q", updated.TTSModel, defaultKittenModel)
	}
	if updated.Voice != defaultKittenVoice {
		t.Fatalf("Voice = %q, want %q", updated.Voice, defaultKittenVoice)
	}
}

func TestApplyKittenConfigPreservesValidSelections(t *testing.T) {
	cfg := config.Default()
	cfg.Voice = "Bella"
	cfg.TTSModel = "KittenML/kitten-tts-micro-0.8"

	updated := applyKittenConfig(cfg, "", "", "/tmp/kitten/bin/python")

	if updated.TTSModel != "KittenML/kitten-tts-micro-0.8" {
		t.Fatalf("TTSModel = %q, want KittenML/kitten-tts-micro-0.8", updated.TTSModel)
	}
	if updated.Voice != "Bella" {
		t.Fatalf("Voice = %q, want Bella", updated.Voice)
	}
}

func TestApplySupertonicConfigUsesSafeDefaults(t *testing.T) {
	cfg := config.Default()
	cfg.Voice = "Yuna"

	updated := applySupertonicConfig(cfg, "", "", "/tmp/supertonic/bin/python")

	if updated.TTSEngine != engineSupertonic {
		t.Fatalf("TTSEngine = %q, want %q", updated.TTSEngine, engineSupertonic)
	}
	if updated.TTSPythonCmd != "/tmp/supertonic/bin/python" {
		t.Fatalf("TTSPythonCmd = %q, want /tmp/supertonic/bin/python", updated.TTSPythonCmd)
	}
	if updated.TTSModel != defaultSupertonicModel {
		t.Fatalf("TTSModel = %q, want %q", updated.TTSModel, defaultSupertonicModel)
	}
	if updated.Voice != defaultSupertonicVoice {
		t.Fatalf("Voice = %q, want %q", updated.Voice, defaultSupertonicVoice)
	}
}

func TestApplySupertonicConfigPreservesValidSelections(t *testing.T) {
	cfg := config.Default()
	cfg.Voice = "F2"
	cfg.TTSModel = "Supertone/supertonic-2"

	updated := applySupertonicConfig(cfg, "", "", "/tmp/supertonic/bin/python")

	if updated.TTSModel != "Supertone/supertonic-2" {
		t.Fatalf("TTSModel = %q, want Supertone/supertonic-2", updated.TTSModel)
	}
	if updated.Voice != "F2" {
		t.Fatalf("Voice = %q, want F2", updated.Voice)
	}
}

func TestSupportsKittenPythonVersion(t *testing.T) {
	if !supportsKittenPythonVersion(pythonVersion{major: 3, minor: 10}) {
		t.Fatal("Python 3.10 should be treated as supported")
	}
	if supportsKittenPythonVersion(pythonVersion{major: 3, minor: 14}) {
		t.Fatal("Python 3.14 should be treated as unsupported")
	}
	if supportsKittenPythonVersion(pythonVersion{major: 2, minor: 7}) {
		t.Fatal("Python 2.7 should be treated as unsupported")
	}
}

func TestSelectCompatiblePythonPrefersNewestSupported(t *testing.T) {
	got, ok := selectCompatiblePython([]pythonCandidate{
		{command: "python3", version: pythonVersion{major: 3, minor: 14}},
		{command: "python3.10", version: pythonVersion{major: 3, minor: 10}},
		{command: "python3.9", version: pythonVersion{major: 3, minor: 9}},
	})
	if !ok {
		t.Fatal("expected to find a compatible Python candidate")
	}
	if got.command != "python3.10" {
		t.Fatalf("selected command = %q, want python3.10", got.command)
	}
}

func TestSelectCompatiblePythonReturnsFalseWhenUnavailable(t *testing.T) {
	_, ok := selectCompatiblePython([]pythonCandidate{
		{command: "python3", version: pythonVersion{major: 3, minor: 14}},
	})
	if ok {
		t.Fatal("expected no compatible Python candidate")
	}
}

func TestApplySayConfigRestoresDefaults(t *testing.T) {
	cfg := config.Default()
	cfg.TTSEngine = engineKittenTTS
	cfg.TTSPythonCmd = "/tmp/kitten/bin/python"
	cfg.TTSModel = "KittenML/kitten-tts-micro-0.8"
	cfg.Voice = "Bella"

	updated := applySayConfig(cfg)
	defaults := config.Default()

	if updated.TTSEngine != engineSay {
		t.Fatalf("TTSEngine = %q, want %q", updated.TTSEngine, engineSay)
	}
	if updated.TTSPythonCmd != defaults.TTSPythonCmd {
		t.Fatalf("TTSPythonCmd = %q, want %q", updated.TTSPythonCmd, defaults.TTSPythonCmd)
	}
	if updated.TTSModel != defaults.TTSModel {
		t.Fatalf("TTSModel = %q, want %q", updated.TTSModel, defaults.TTSModel)
	}
	if updated.Voice != defaults.Voice {
		t.Fatalf("Voice = %q, want %q", updated.Voice, defaults.Voice)
	}
}

func TestShouldResetKittenConfigByEngine(t *testing.T) {
	cfg := config.Default()
	cfg.TTSEngine = engineKittenTTS

	if !shouldResetAfterKittenUninstall(cfg) {
		t.Fatal("expected uninstall to reset config when engine is kittentts")
	}
}

func TestShouldResetKittenConfigByManagedPythonPath(t *testing.T) {
	cfg := config.Default()
	cfg.TTSEngine = engineSay
	cfg.TTSPythonCmd = kittenVenvPythonPath(kittenInstallDir())

	if !shouldResetAfterKittenUninstall(cfg) {
		t.Fatal("expected uninstall to reset config when managed python path is active")
	}
}

func TestShouldNotResetUnrelatedSayConfig(t *testing.T) {
	cfg := config.Default()
	cfg.TTSEngine = engineSay
	cfg.Voice = "Samantha"
	cfg.TTSPythonCmd = "/opt/homebrew/bin/python3.10"

	if shouldResetAfterKittenUninstall(cfg) {
		t.Fatal("did not expect uninstall to reset unrelated say config")
	}
}

func TestShouldResetSupertonicConfigByEngine(t *testing.T) {
	cfg := config.Default()
	cfg.TTSEngine = engineSupertonic

	if !shouldResetAfterSupertonicUninstall(cfg) {
		t.Fatal("expected uninstall to reset config when engine is supertonic")
	}
}

func TestShouldResetSupertonicConfigByManagedPythonPath(t *testing.T) {
	cfg := config.Default()
	cfg.TTSEngine = engineSay
	cfg.TTSPythonCmd = supertonicVenvPythonPath(supertonicInstallDir())

	if !shouldResetAfterSupertonicUninstall(cfg) {
		t.Fatal("expected uninstall to reset config when managed python path is active")
	}
}
