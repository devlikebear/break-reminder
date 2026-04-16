package tts

import (
	"fmt"
	"os/exec"
	"strings"
	"unicode"
)

const (
	engineSay              = "say"
	engineKittenTTS        = "kittentts"
	engineSupertonic       = "supertonic"
	engineGemini           = "gemini"
	defaultKittenModel     = "KittenML/kitten-tts-nano-0.8"
	defaultSupertonicModel = "Supertone/supertonic-2"
	defaultGeminiModel     = "gemini-3.1-flash-tts-preview"
	defaultPythonCommand   = "python3"
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

var supertonicVoices = map[string]string{
	"f1": "F1",
	"f2": "F2",
	"f3": "F3",
	"f4": "F4",
	"f5": "F5",
	"m1": "M1",
	"m2": "M2",
	"m3": "M3",
	"m4": "M4",
	"m5": "M5",
}

var geminiVoices = map[string]string{
	"zephyr":        "Zephyr",
	"puck":          "Puck",
	"charon":        "Charon",
	"kore":          "Kore",
	"fenrir":        "Fenrir",
	"leda":          "Leda",
	"orus":          "Orus",
	"aoede":         "Aoede",
	"callirrhoe":    "Callirrhoe",
	"autonoe":       "Autonoe",
	"enceladus":     "Enceladus",
	"iapetus":       "Iapetus",
	"umbriel":       "Umbriel",
	"algieba":       "Algieba",
	"despina":       "Despina",
	"erinome":       "Erinome",
	"algenib":       "Algenib",
	"rasalgethi":    "Rasalgethi",
	"laomedeia":     "Laomedeia",
	"achernar":      "Achernar",
	"alnilam":       "Alnilam",
	"schedar":       "Schedar",
	"gacrux":        "Gacrux",
	"pulcherrima":   "Pulcherrima",
	"achird":        "Achird",
	"zubenelgenubi": "Zubenelgenubi",
	"vindemiatrix":  "Vindemiatrix",
	"sadachbia":     "Sadachbia",
	"sadaltager":    "Sadaltager",
	"sulafat":       "Sulafat",
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

const supertonicTTSPythonScript = `
import os
import subprocess
import sys
import tempfile

from supertonic import TTS

voice, lang, message = sys.argv[1], sys.argv[2], sys.argv[3]
fd, output_path = tempfile.mkstemp(suffix=".wav")
os.close(fd)

try:
    tts = TTS(auto_download=True)
    style = tts.get_voice_style(voice_name=voice)
    wav, _ = tts.synthesize(message, voice_style=style, lang=lang)
    tts.save_audio(wav, output_path)
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
	case engineSupertonic:
		return engineSupertonic
	case engineGemini:
		return engineGemini
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

func normalizeModelForEngine(engine, model string) string {
	switch normalizeEngine(engine) {
	case engineKittenTTS:
		return normalizeKittenModel(model)
	case engineSupertonic:
		return normalizeSupertonicModel(model)
	case engineGemini:
		return normalizeGeminiModel(model)
	default:
		return strings.TrimSpace(model)
	}
}

func normalizePythonCommand(pythonCmd string) string {
	if strings.TrimSpace(pythonCmd) == "" {
		return defaultPythonCommand
	}
	return strings.TrimSpace(pythonCmd)
}

func normalizeSupertonicModel(model string) string {
	if strings.TrimSpace(model) == "" {
		return defaultSupertonicModel
	}
	return strings.TrimSpace(model)
}

func kittenVoiceAvailable(voice string) bool {
	_, ok := canonicalKittenVoice(voice)
	return ok
}

func canonicalKittenVoice(voice string) (string, bool) {
	canonical, ok := kittenVoices[strings.ToLower(strings.TrimSpace(voice))]
	return canonical, ok
}

func supertonicVoiceAvailable(voice string) bool {
	_, ok := canonicalSupertonicVoice(voice)
	return ok
}

func canonicalSupertonicVoice(voice string) (string, bool) {
	canonical, ok := supertonicVoices[strings.ToLower(strings.TrimSpace(voice))]
	return canonical, ok
}

func normalizeGeminiModel(model string) string {
	if strings.TrimSpace(model) == "" {
		return defaultGeminiModel
	}
	return strings.TrimSpace(model)
}

func geminiVoiceAvailable(voice string) bool {
	_, ok := canonicalGeminiVoice(voice)
	return ok
}

func canonicalGeminiVoice(voice string) (string, bool) {
	canonical, ok := geminiVoices[strings.ToLower(strings.TrimSpace(voice))]
	return canonical, ok
}

func detectSupertonicLanguage(message string) string {
	for _, r := range message {
		if unicode.In(r, unicode.Hangul) {
			return "ko"
		}
	}
	return "en"
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
	case engineSupertonic:
		resolvedVoice := voice
		if canonical, ok := canonicalSupertonicVoice(voice); ok {
			resolvedVoice = canonical
		}
		return exec.Command(
			normalizePythonCommand(pythonCmd),
			"-c",
			supertonicTTSPythonScript,
			resolvedVoice,
			detectSupertonicLanguage(message),
			message,
		), nil
	default:
		return nil, fmt.Errorf("unsupported TTS engine %q", strings.TrimSpace(engine))
	}
}
