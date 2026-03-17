package tts

// Speaker provides text-to-speech.
type Speaker interface {
	Speak(voice, message string) error
	Available(voice string) bool
}
