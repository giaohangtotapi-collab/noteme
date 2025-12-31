package ai

import (
	"fmt"
	"strings"
)

// BuildPrompt builds the complete prompt for LLM
func BuildPrompt(transcript string, context string) (string, string) {
	systemPrompt := `You are an assistant that analyzes Vietnamese speech transcripts.
You must be precise, neutral, and factual.
Do not invent information.
Only use information explicitly present in the transcript.
Return valid JSON only.
You MUST fill all fields in the response, even if some are empty arrays.`

	userPrompt := fmt.Sprintf(`Transcript:
"""
%s
"""

Context: %s

Tasks:
1. Write a concise summary (max 5 bullet points) - REQUIRED, must be an array of strings.
2. Extract clear action items, if any - REQUIRED, must be an array of strings (can be empty if none).
3. Extract key facts, numbers, names, or commitments - REQUIRED, must be an array of strings (can be empty if none).
4. Create a brief summary for Zalo (3 bullet points max) - REQUIRED, must be a string (can be empty if no content).

IMPORTANT RULES:
- ALL fields are REQUIRED in the JSON response.
- summary: array of strings, at least 1 item if transcript has content
- action_items: array of strings, can be empty [] if no actions found
- key_points: array of strings, extract important facts/numbers/names/commitments, can be empty [] if none
- zalo_brief: string, 3 bullet points format like "- Point 1\n- Point 2\n- Point 3", can be empty string if no content
- If transcript is about lecture/thinking, key_points should contain main ideas/concepts
- If transcript is about meeting, action_items should contain tasks/commitments

Output JSON exactly in the following format (ALL fields required, use empty array [] or empty string "" if no data):

{
  "context": "%s",
  "summary": ["point 1", "point 2"],
  "action_items": ["task 1", "task 2"],
  "key_points": ["fact 1", "fact 2"],
  "zalo_brief": "- Point 1\\n- Point 2\\n- Point 3"
}

CRITICAL: You MUST provide all fields:
- summary: MUST have at least 1 item if transcript has meaningful content
- action_items: array (can be empty [] if no actions)
- key_points: array (MUST extract key facts/numbers/names/ideas, can be empty [] only if truly no key information)
- zalo_brief: string (MUST provide 3 bullet points format, use empty string "" only if transcript is completely empty)`, transcript, context, context)

	return systemPrompt, userPrompt
}

// BuildPromptV1 builds prompt according to NoteMe Prompt Engine v1 spec
func BuildPromptV1(transcript string) (string, string) {
	systemPrompt := `You are NoteMe's AI brain - an advanced assistant for Vietnamese users. 
Your task is to read the transcript and perform 2 steps: (1) Classify context, (2) Present results in the required structure.

PRINCIPLES FOR VIETNAMESE PROCESSING:
1. Vinglish: Keep words like: Approve, Deadline, Task, KPI, Pitching, Workshop, Follow-up, Feedback...
2. Addressing: Use professional titles (Anh/Chị/Bạn or proper names). Never use "Tôi" and "Bạn" like machine translation.
3. Filter noise: Remove 100% greeting sentences, mic testing, ordering drinks, casual chat.`

	userPrompt := fmt.Sprintf(`Analyze this Vietnamese transcript:

"""
%s
"""

STEP 1: CONTEXT CLASSIFICATION
Classify the content as:
- MEETING: Multiple people discussing, task assignments, decisions made
- THINKING: One person speaking, self-reflection, scattered ideas
- LECTURE: One person speaking, systematic content, educational

STEP 2: OUTPUT STRUCTURE
Return ONLY valid JSON (no extra text):

{
  "context": "MEETING | THINKING | LECTURE",
  "confidence_score": 0.0,
  "content": {
    "summary": "Short paragraph 3-5 sentences summarizing main content.",
    "action_items": [
      {"task": "Task name", "assignee": "Person/Department", "deadline": "If any"}
    ],
    "key_ideas": [
      "Most important idea or information 1",
      "Most important idea or information 2"
    ]
  },
  "zalo_brief": "Very short summary (3 bullet points) for quick copy-paste."
}`, transcript)

	return systemPrompt, userPrompt
}

// CleanTranscript removes noise from transcript
func CleanTranscript(transcript string) string {
	// Remove common noise patterns
	noisePatterns := []string{
		"xin chào", "chào bạn", "hello", "hi",
		"test mic", "thử mic", "check mic",
		"được rồi", "ok", "okay", "ừ", "ừm",
	}

	cleaned := transcript
	for _, pattern := range noisePatterns {
		cleaned = strings.ReplaceAll(strings.ToLower(cleaned), pattern, "")
	}

	// Remove extra whitespace
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.ReplaceAll(cleaned, "  ", " ")

	return cleaned
}
