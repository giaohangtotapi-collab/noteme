package stt

// Result represents the result of a speech-to-text transcription
type Result struct {
	Transcript  string  // The transcribed text
	Confidence  float64 // Confidence score (0.0-1.0), may be 0 if not provided
	Provider    string  // The provider used (e.g., "fpt", "google")
	RawResponse string  // Raw response from the provider (for debugging/logging)
}
