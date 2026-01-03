package ai

import (
	"fmt"
	"strings"
)

// BuildPrompt builds the complete prompt for LLM
func BuildPrompt(transcript string, context string) (string, string) {
	systemPrompt := `Bạn là trợ lý AI phân tích bản ghi âm tiếng Việt cho NoteMe.
Bạn phải chính xác, trung lập và dựa trên sự thật.
KHÔNG được bịa đặt thông tin.
CHỈ sử dụng thông tin có trong transcript.
Trả về JSON hợp lệ.
BẮT BUỘC điền đầy đủ tất cả các trường, kể cả nếu một số là mảng rỗng.

QUAN TRỌNG VỀ NGÔN NGỮ:
- TẤT CẢ nội dung phải bằng TIẾNG VIỆT
- CHỈ giữ lại keywords chuyên ngành bằng tiếng Anh (Vinglish) như: API, Backend, Frontend, MVP, STT, AI, OpenAI, FPT.AI, Golang, Flutter, React Native, Firebase, Deadline, Task, KPI, Meeting, Call, Share, Mindmap, Demo, Test, Dev, Developer, etc.
- KHÔNG dịch các thuật ngữ chuyên ngành sang tiếng Việt
- Tất cả các câu, đoạn văn khác phải bằng tiếng Việt hoàn toàn`

	userPrompt := fmt.Sprintf(`Transcript:
"""
%s
"""

Context: %s

Nhiệm vụ:
1. Viết tóm tắt ngắn gọn (tối đa 5 điểm) - BẮT BUỘC, phải là mảng các chuỗi tiếng Việt.
2. Trích xuất action items rõ ràng, nếu có - BẮT BUỘC, phải là mảng các chuỗi tiếng Việt (có thể rỗng nếu không có).
3. Trích xuất các sự kiện quan trọng, số liệu, tên, hoặc cam kết - BẮT BUỘC, phải là mảng các chuỗi tiếng Việt (có thể rỗng nếu không có).
4. Tạo tóm tắt ngắn cho Zalo (tối đa 3 điểm) - BẮT BUỘC, phải là chuỗi tiếng Việt (có thể rỗng nếu không có nội dung).

QUY TẮC QUAN TRỌNG:
- TẤT CẢ các trường đều BẮT BUỘC trong JSON response.
- summary: mảng các chuỗi tiếng Việt, ít nhất 1 mục nếu transcript có nội dung
- action_items: mảng các chuỗi tiếng Việt, có thể rỗng [] nếu không tìm thấy action
- key_points: mảng các chuỗi tiếng Việt, trích xuất các sự kiện/số liệu/tên/cam kết quan trọng, có thể rỗng [] nếu không có
- zalo_brief: chuỗi tiếng Việt, định dạng 3 điểm như "- Điểm 1\n- Điểm 2\n- Điểm 3", có thể là chuỗi rỗng "" nếu không có nội dung
- Nếu transcript về lecture/thinking, key_points nên chứa các ý tưởng/khái niệm chính
- Nếu transcript về meeting, action_items nên chứa các nhiệm vụ/cam kết
- TẤT CẢ nội dung phải bằng TIẾNG VIỆT, chỉ giữ keywords chuyên ngành bằng tiếng Anh (API, Backend, MVP, etc.)

Trả về JSON chính xác theo format sau (TẤT CẢ các trường bắt buộc, dùng mảng rỗng [] hoặc chuỗi rỗng "" nếu không có dữ liệu):

{
  "context": "%s",
  "summary": ["điểm 1", "điểm 2"],
  "action_items": ["nhiệm vụ 1", "nhiệm vụ 2"],
  "key_points": ["sự kiện 1", "sự kiện 2"],
  "zalo_brief": "- Điểm 1\\n- Điểm 2\\n- Điểm 3"
}

QUAN TRỌNG: Bạn PHẢI cung cấp tất cả các trường:
- summary: PHẢI có ít nhất 1 mục nếu transcript có nội dung ý nghĩa
- action_items: mảng (có thể rỗng [] nếu không có actions)
- key_points: mảng (PHẢI trích xuất các sự kiện/số liệu/tên/ý tưởng quan trọng, chỉ rỗng [] nếu thực sự không có thông tin quan trọng)
- zalo_brief: chuỗi (PHẢI cung cấp định dạng 3 điểm, chỉ dùng chuỗi rỗng "" nếu transcript hoàn toàn trống)
- TẤT CẢ nội dung phải bằng TIẾNG VIỆT, chỉ giữ keywords chuyên ngành bằng tiếng Anh`, transcript, context, context)

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
