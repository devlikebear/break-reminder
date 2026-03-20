//go:build darwin

package tts

import (
	"fmt"
	"os/exec"
	"strings"
)

type DarwinSpeaker struct {
	engine    string
	model     string
	pythonCmd string
}

func NewSpeaker(engine, model, pythonCmd string) Speaker {
	return &DarwinSpeaker{
		engine:    normalizeEngine(engine),
		model:     normalizeKittenModel(model),
		pythonCmd: normalizePythonCommand(pythonCmd),
	}
}

func (s *DarwinSpeaker) Speak(voice, message string) error {
	if err := s.validate(voice); err != nil {
		return err
	}

	cmd, err := buildSpeakCommand(s.engine, s.model, s.pythonCmd, voice, message)
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
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
	cmd := exec.Command(
		pythonCmd,
		"-c",
		`import importlib.util, sys; sys.exit(0 if importlib.util.find_spec("kittentts") else 1)`,
	)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
