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
