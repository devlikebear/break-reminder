package tts

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	engineSay            = "say"
	engineKittenTTS      = "kittentts"
	defaultKittenModel   = "KittenML/kitten-tts-nano-0.8"
	defaultPythonCommand = "python3"
)

var kittenVoices = map[string]string{
	"bella":  "Bella",
	"bruno":  "Bruno",
	"hugo":   "Hugo",
	"jasper": "Jasper",
	"kiki":   "Kiki",
	"leo":    "Leo",
	"luna":   "Luna",
	"rosie":  "Rosie",
}

const kittenTTSPythonScript = `
import os
import subprocess
import sys
import tempfile

from kittentts import KittenTTS

model_name, voice, message = sys.argv[1], sys.argv[2], sys.argv[3]
fd, output_path = tempfile.mkstemp(suffix=".wav")
os.close(fd)

try:
    model = KittenTTS(model_name)
    model.generate_to_file(message, output_path, voice=voice)
    subprocess.run(["afplay", output_path], check=True)
finally:
    try:
        os.remove(output_path)
    except OSError:
        pass
`

func normalizeEngine(engine string) string {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "", engineSay:
		return engineSay
	case "kitten", engineKittenTTS:
		return engineKittenTTS
	default:
		return strings.ToLower(strings.TrimSpace(engine))
	}
}

func normalizeKittenModel(model string) string {
	if strings.TrimSpace(model) == "" {
		return defaultKittenModel
	}
	return strings.TrimSpace(model)
}

func normalizePythonCommand(pythonCmd string) string {
	if strings.TrimSpace(pythonCmd) == "" {
		return defaultPythonCommand
	}
	return strings.TrimSpace(pythonCmd)
}

func kittenVoiceAvailable(voice string) bool {
	_, ok := canonicalKittenVoice(voice)
	return ok
}

func canonicalKittenVoice(voice string) (string, bool) {
	canonical, ok := kittenVoices[strings.ToLower(strings.TrimSpace(voice))]
	return canonical, ok
}

func buildSpeakCommand(engine, model, pythonCmd, voice, message string) (*exec.Cmd, error) {
	switch normalizeEngine(engine) {
	case engineSay:
		return exec.Command("say", "-v", voice, message), nil
	case engineKittenTTS:
		resolvedVoice := voice
		if canonical, ok := canonicalKittenVoice(voice); ok {
			resolvedVoice = canonical
		}
		return exec.Command(
			normalizePythonCommand(pythonCmd),
			"-c",
			kittenTTSPythonScript,
			normalizeKittenModel(model),
			resolvedVoice,
			message,
		), nil
	default:
		return nil, fmt.Errorf("unsupported TTS engine %q", strings.TrimSpace(engine))
	}
}
