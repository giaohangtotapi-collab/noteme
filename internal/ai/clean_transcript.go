package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

// CleanedTranscriptResult represents the cleaned transcript result
type CleanedTranscriptResult struct {
	CleanedText  string   `json:"cleaned_text"`
	Summary      string   `json:"summary"`
	DecodedWords []string `json:"decoded_words,omitempty"`
}

// CleanTranscriptWithAI cleans and minimizes transcript using OpenAI
func CleanTranscriptWithAI(transcript string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	log.Printf("=== Cleaning Transcript with AI ===")
	log.Printf("Original transcript length: %d characters", len(transcript))

	// Build prompt according to promt_ai_1.md with enhanced context understanding
	systemPrompt := `Bạn là một AI chuyên phân tích hội thoại tiếng Việt trong lĩnh vực công nghệ/startup, có khả năng:
- Suy luận từ lời nói không rõ
- Sửa lỗi nghe sai, nói lắp, nói nhanh
- Hiểu thuật ngữ kỹ thuật, tiếng lóng, từ mượn tiếng Anh (Vinglish)
- Nhận diện và sửa tên riêng, tên dự án, tên công nghệ bị nhận dạng sai
- Phục hồi nội dung hội thoại về dạng rõ ràng, đúng ý người nói

KIẾN THỨC VỀ CÔNG NGHỆ:
- Ngôn ngữ lập trình: Golang, Python, JavaScript, TypeScript, Java, C++, etc.
- Framework/Platform: React, Vue, Angular, Flutter, React Native, Node.js, etc.
- AI/ML: OpenAI, GPT, Claude, FPT.AI, Speech-to-Text, STT, etc.
- Thuật ngữ: API, Backend, Frontend, MVP, Demo, Test, Dev, Developer, etc.
- Vinglish phổ biến: App, Task, Deadline, KPI, Meeting, Call, Share, Mindmap, etc.

NGUYÊN TẮC:
- Không suy diễn quá mức
- Không "làm đẹp" nội dung ngoài ý người nói
- Giữ nguyên ý định gốc, không thêm ý cá nhân
- Ưu tiên sửa các từ kỹ thuật, tên riêng, Vinglish bị nhận dạng sai

QUAN TRỌNG VỀ NGÔN NGỮ:
- TẤT CẢ output phải bằng TIẾNG VIỆT
- CHỈ giữ lại keywords chuyên ngành bằng tiếng Anh (Vinglish) như: API, Backend, Frontend, MVP, STT, AI, OpenAI, FPT.AI, Golang, Flutter, React Native, Firebase, Deadline, Task, KPI, Meeting, Call, Share, Mindmap, Demo, Test, Dev, Developer, etc.
- KHÔNG dịch các thuật ngữ chuyên ngành sang tiếng Việt
- cleaned_text và summary phải bằng tiếng Việt hoàn toàn, chỉ giữ keywords chuyên ngành`

	userPrompt := fmt.Sprintf(`Hãy phân tích và làm sạch đoạn hội thoại sau (đã được chuyển từ âm thanh sang text, có thể có nhiều lỗi nhận dạng):

"""
%s
"""

Thực hiện các bước CHI TIẾT:

BƯỚC 1 - Hiểu ngữ cảnh:
- Xác định chủ đề (công nghệ/startup/dự án/phát triển phần mềm)
- Xác định mục đích người nói (trao đổi công việc, giao việc, thảo luận kỹ thuật, planning)

BƯỚC 2 - Giải mã từ nghe sai (QUAN TRỌNG):
- Tên riêng/Tên dự án: "Nút Mi" có thể là "NoteMe", "Pulse" có thể là tên feature
- Thuật ngữ kỹ thuật: "Control Back" → "Golang", "FPT A" → "FPT.AI"
- Vinglish bị nhận dạng sai: "credit" → "Vinglish", "xe" → "share", "internet" → "mindmap"
- Từ tiếng Anh: "Anderson" → "Hold", "Update" → "Ask", "để mua" → "Demo"
- Cụm từ: "Trí thông minh điện tử" → "hàng nội địa", "đổi dev" → "đội Dev"
- Từ lóng: "pro" → "bro", "tư vấn" → "test"

BƯỚC 3 - Viết lại nội dung:
- Câu đầy đủ, có dấu câu, ngữ pháp đúng
- Giữ nguyên phong cách nói (thân mật/chuyên nghiệp)
- Sửa tất cả lỗi nhận dạng đã phát hiện

BƯỚC 4 - Tóm tắt:
- Mục tiêu chính, yêu cầu/deadline, quyết định quan trọng

Trả về JSON với format:
{
  "cleaned_text": "Bản viết lại rõ ràng, chuẩn, đã sửa TẤT CẢ lỗi nhận dạng, bằng TIẾNG VIỆT",
  "summary": "Tóm tắt ngắn gọn bằng TIẾNG VIỆT",
  "decoded_words": ["từ sai → từ đúng", "từ sai → từ đúng"]
}

QUAN TRỌNG:
- cleaned_text: PHẢI sửa tất cả lỗi nhận dạng, đặc biệt là tên riêng, thuật ngữ kỹ thuật, Vinglish. PHẢI bằng TIẾNG VIỆT, chỉ giữ keywords chuyên ngành bằng tiếng Anh
- summary: PHẢI bằng TIẾNG VIỆT, chỉ giữ keywords chuyên ngành bằng tiếng Anh
- decoded_words: Liệt kê các từ/cụm từ đã sửa theo format "sai → đúng"
- Dựa vào ngữ cảnh để suy đoán hợp lý (ví dụ: nếu nói về app, "Nút Mi" rất có thể là "NoteMe")
- Nếu không chắc chắn, ưu tiên giữ nguyên nhưng ghi chú trong decoded_words
- TẤT CẢ nội dung phải bằng TIẾNG VIỆT, chỉ giữ keywords chuyên ngành bằng tiếng Anh`, transcript)

	// Create OpenAI client
	client := openai.NewClient(apiKey)

	// Call OpenAI API
	ctx := context.Background()
	log.Printf("Calling OpenAI API to clean transcript...")

	req := openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.2, // Very low temperature for accurate cleaning
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Printf("OpenAI API error while cleaning: %v", err)
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI returned no choices")
	}

	content := resp.Choices[0].Message.Content
	log.Printf("OpenAI cleaning response received (length: %d)", len(content))
	log.Printf("Usage - Prompt tokens: %d, Completion tokens: %d, Total tokens: %d",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

	// Parse JSON response
	var result CleanedTranscriptResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("Failed to parse cleaning response. Attempting to extract from markdown...")
		extractedContent := extractJSONFromMarkdown(content)
		if err := json.Unmarshal([]byte(extractedContent), &result); err != nil {
			log.Printf("ERROR: Failed to parse cleaned transcript JSON. Raw: %s", content)
			return "", fmt.Errorf("failed to parse OpenAI response as JSON: %w", err)
		}
	}

	log.Printf("=== Transcript Cleaning Complete ===")
	log.Printf("Cleaned text length: %d characters", len(result.CleanedText))
	log.Printf("Summary: %s", result.Summary)
	if len(result.DecodedWords) > 0 {
		log.Printf("Decoded words: %v", result.DecodedWords)
	}

	// Return cleaned text
	if result.CleanedText == "" {
		log.Printf("WARNING: Cleaned text is empty, using original transcript")
		return transcript, nil
	}

	return result.CleanedText, nil
}
