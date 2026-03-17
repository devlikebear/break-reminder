//go:build !darwin

package tts

import "fmt"

type StubSpeaker struct{}

func NewSpeaker() Speaker {
	return &StubSpeaker{}
}

func (s *StubSpeaker) Speak(voice, message string) error {
	fmt.Printf("[tts] %s: %s\n", voice, message)
	return nil
}

func (s *StubSpeaker) Available(voice string) bool {
	return false
}
