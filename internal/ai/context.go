package ai

import (
	"strings"
)

// DetectContext detects context type based on simple rules
// Returns: "meeting", "lecture", or "thinking"
func DetectContext(transcript string) string {
	transcript = strings.ToLower(transcript)

	// Meeting keywords
	meetingKeywords := []string{
		"họp", "dự án", "deadline", "gửi", "báo cáo",
		"khách hàng", "đồng nghiệp", "team", "nhóm",
		"thống nhất", "chốt", "phê duyệt", "approve",
		"task", "công việc", "nhiệm vụ",
	}

	// Lecture keywords
	lectureKeywords := []string{
		"bài giảng", "thầy", "cô", "chương", "ví dụ",
		"kiến thức", "học", "giải thích", "định nghĩa",
		"khái niệm", "nguyên lý", "phương pháp",
	}

	// Count matches
	meetingCount := 0
	lectureCount := 0

	for _, keyword := range meetingKeywords {
		if strings.Contains(transcript, keyword) {
			meetingCount++
		}
	}

	for _, keyword := range lectureKeywords {
		if strings.Contains(transcript, keyword) {
			lectureCount++
		}
	}

	// Determine context
	if meetingCount > 0 && meetingCount >= lectureCount {
		return "meeting"
	}
	if lectureCount > 0 {
		return "lecture"
	}

	// Default to thinking
	return "thinking"
}
