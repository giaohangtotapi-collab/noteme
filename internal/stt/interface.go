package stt

// Provider defines the interface for speech-to-text providers
type Provider interface {
	// Transcribe transcribes an audio file and returns the result
	Transcribe(audioPath string) (*Result, error)

	// Name returns the name of the provider (e.g., "fpt", "google")
	Name() string
}
