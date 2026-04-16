//go:build darwin

package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type DarwinSpeaker struct {
	engine    string
	model     string
	pythonCmd string
	apiKey    string
}

func NewSpeaker(engine, model, pythonCmd, apiKey string) Speaker {
	return &DarwinSpeaker{
		engine:    normalizeEngine(engine),
		model:     normalizeModelForEngine(engine, model),
		pythonCmd: normalizePythonCommand(pythonCmd),
		apiKey:    strings.TrimSpace(apiKey),
	}
}

func (s *DarwinSpeaker) Speak(voice, message string) error {
	return s.speak(voice, message, false)
}

func (s *DarwinSpeaker) speak(voice, message string, wait bool) error {
	if err := s.validate(voice); err != nil {
		return err
	}

	if s.engine == engineGemini {
		return s.speakGemini(voice, message, wait)
	}

	cmd, err := buildSpeakCommand(s.engine, s.model, s.pythonCmd, voice, message)
	if err != nil {
		return err
	}

	if wait {
		return cmd.Run()
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

func (s *DarwinSpeaker) speakGemini(voice, message string, wait bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), geminiHTTPTimeout+5*time.Second)
	defer cancel()

	path, err := synthesizeGemini(ctx, s.apiKey, s.model, voice, message)
	if err != nil {
		return err
	}

	cmd := exec.Command("afplay", path)
	if wait {
		defer os.Remove(path)
		return cmd.Run()
	}

	if err := cmd.Start(); err != nil {
		_ = os.Remove(path)
		return err
	}
	go func() {
		_ = cmd.Wait()
		_ = os.Remove(path)
	}()
	return nil
}

func (s *DarwinSpeaker) Available(voice string) bool {
	switch s.engine {
	case engineSay:
		return sayVoiceAvailable(voice)
	case engineKittenTTS:
		if !kittenVoiceAvailable(voice) {
			return false
		}
		if _, err := exec.LookPath(s.pythonCmd); err != nil {
			return false
		}
		if _, err := exec.LookPath("afplay"); err != nil {
			return false
		}
		ok, err := kittenModuleInstalled(s.pythonCmd)
		return err == nil && ok
	case engineSupertonic:
		if !supertonicVoiceAvailable(voice) {
			return false
		}
		if _, err := exec.LookPath(s.pythonCmd); err != nil {
			return false
		}
		if _, err := exec.LookPath("afplay"); err != nil {
			return false
		}
		ok, err := supertonicModuleInstalled(s.pythonCmd)
		return err == nil && ok
	case engineGemini:
		if !geminiVoiceAvailable(voice) {
			return false
		}
		if s.apiKey == "" {
			return false
		}
		if _, err := exec.LookPath("afplay"); err != nil {
			return false
		}
		return true
	default:
		return false
	}
}

func (s *DarwinSpeaker) validate(voice string) error {
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return fmt.Errorf("voice is required")
	}

	switch s.engine {
	case engineSay:
		if !sayVoiceAvailable(voice) {
			return fmt.Errorf("voice %q not found for macOS say", voice)
		}
	case engineKittenTTS:
		if !kittenVoiceAvailable(voice) {
			return fmt.Errorf("voice %q not supported by KittenTTS", voice)
		}
		if _, err := exec.LookPath(s.pythonCmd); err != nil {
			return fmt.Errorf("python command %q not found", s.pythonCmd)
		}
		if _, err := exec.LookPath("afplay"); err != nil {
			return fmt.Errorf("afplay command not found")
		}
		ok, err := kittenModuleInstalled(s.pythonCmd)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("kittentts is not installed for %s", s.pythonCmd)
		}
	case engineSupertonic:
		if !supertonicVoiceAvailable(voice) {
			return fmt.Errorf("voice %q not supported by Supertonic", voice)
		}
		if _, err := exec.LookPath(s.pythonCmd); err != nil {
			return fmt.Errorf("python command %q not found", s.pythonCmd)
		}
		if _, err := exec.LookPath("afplay"); err != nil {
			return fmt.Errorf("afplay command not found")
		}
		ok, err := supertonicModuleInstalled(s.pythonCmd)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("supertonic is not installed for %s", s.pythonCmd)
		}
	case engineGemini:
		if !geminiVoiceAvailable(voice) {
			return fmt.Errorf("voice %q not supported by Gemini TTS", voice)
		}
		if s.apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY is not set; export env or set tts_api_key in config")
		}
		if _, err := exec.LookPath("afplay"); err != nil {
			return fmt.Errorf("afplay command not found")
		}
	default:
		return fmt.Errorf("unsupported TTS engine %q", s.engine)
	}

	return nil
}

func sayVoiceAvailable(voice string) bool {
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return false
	}

	out, err := exec.Command("say", "-v", "?").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, voice+" ") {
			return true
		}
	}
	return false
}

func kittenModuleInstalled(pythonCmd string) (bool, error) {
	return pythonModuleInstalled(pythonCmd, "kittentts")
}

func supertonicModuleInstalled(pythonCmd string) (bool, error) {
	return pythonModuleInstalled(pythonCmd, "supertonic")
}

func pythonModuleInstalled(pythonCmd, module string) (bool, error) {
	cmd := exec.Command(
		pythonCmd,
		"-c",
		fmt.Sprintf(`import importlib.util, sys; sys.exit(0 if importlib.util.find_spec(%q) else 1)`, module),
	)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func VoiceAvailable(engine, model, pythonCmd, apiKey, voice string) bool {
	return NewSpeaker(engine, model, pythonCmd, apiKey).Available(voice)
}

func SpeakAndWait(engine, model, pythonCmd, apiKey, voice, message string) error {
	speaker := &DarwinSpeaker{
		engine:    normalizeEngine(engine),
		model:     normalizeModelForEngine(engine, model),
		pythonCmd: normalizePythonCommand(pythonCmd),
		apiKey:    strings.TrimSpace(apiKey),
	}
	return speaker.speak(voice, message, true)
}
