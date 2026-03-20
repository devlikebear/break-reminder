//go:build !darwin

package tts

import "fmt"

type StubSpeaker struct {
	engine string
}

func NewSpeaker(engine, model, pythonCmd string) Speaker {
	return &StubSpeaker{engine: normalizeEngine(engine)}
}

func (s *StubSpeaker) Speak(voice, message string) error {
	fmt.Printf("[tts:%s] %s: %s\n", s.engine, voice, message)
	return nil
}

func (s *StubSpeaker) Available(voice string) bool {
	return false
}

func VoiceAvailable(engine, model, pythonCmd, voice string) bool {
	return NewSpeaker(engine, model, pythonCmd).Available(voice)
}

func SpeakAndWait(engine, model, pythonCmd, voice, message string) error {
	return NewSpeaker(engine, model, pythonCmd).Speak(voice, message)
}
