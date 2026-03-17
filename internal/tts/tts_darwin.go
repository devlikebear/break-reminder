//go:build darwin

package tts

import (
	"os/exec"
	"strings"
)

type DarwinSpeaker struct{}

func NewSpeaker() Speaker {
	return &DarwinSpeaker{}
}

func (s *DarwinSpeaker) Speak(voice, message string) error {
	cmd := exec.Command("say", "-v", voice, message)
	return cmd.Start() // non-blocking, like bash's `say ... &`
}

func (s *DarwinSpeaker) Available(voice string) bool {
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
