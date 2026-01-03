package storage

import (
	"noteme/internal/ai"
	"sync"
)

var (
	analyses   = make(map[string]*ai.AnalysisResult)
	muAnalysis sync.Mutex
)

// SaveAnalysis saves analysis result for a recording
func SaveAnalysis(recordingID string, result *ai.AnalysisResult) {
	muAnalysis.Lock()
	defer muAnalysis.Unlock()
	analyses[recordingID] = result
}

// GetAnalysis retrieves analysis result for a recording
func GetAnalysis(recordingID string) (*ai.AnalysisResult, bool) {
	muAnalysis.Lock()
	defer muAnalysis.Unlock()
	result, ok := analyses[recordingID]
	if !ok {
		return nil, false
	}
	// Return a copy to avoid race conditions
	resultCopy := *result
	return &resultCopy, true
}

// GetAllAnalyses retrieves all analysis results
func GetAllAnalyses() map[string]*ai.AnalysisResult {
	muAnalysis.Lock()
	defer muAnalysis.Unlock()

	// Return a copy of the map
	result := make(map[string]*ai.AnalysisResult)
	for k, v := range analyses {
		resultCopy := *v
		result[k] = &resultCopy
	}
	return result
}
