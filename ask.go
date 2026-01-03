package ai

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// AskAnything answers questions based on all analyzed data
func AskAnything(question string, allAnalyses []AnalysisContext) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	if len(allAnalyses) == 0 {
		return "", fmt.Errorf("no analysis data available to answer the question")
	}

	log.Printf("=== Ask Anything Request ===")
	log.Printf("Question: %s", question)
	log.Printf("Number of analyses: %d", len(allAnalyses))

	// Build context from all analyses
	contextText := buildContextFromAnalyses(allAnalyses)
	log.Printf("Context length: %d characters", len(contextText))

	// Build prompt
	systemPrompt := `Bạn là trợ lý AI của NoteMe. Nhiệm vụ của bạn là trả lời câu hỏi dựa trên dữ liệu đã được phân tích từ các cuộc ghi âm.

NGUYÊN TẮC:
- Chỉ trả lời dựa trên thông tin có trong dữ liệu được cung cấp
- Không bịa đặt thông tin
- Nếu không có thông tin, hãy nói rõ "Không tìm thấy thông tin trong dữ liệu đã ghi"
- Trả lời ngắn gọn, rõ ràng, bằng tiếng Việt
- Không chat dài, không roleplay, chỉ trả lời trực tiếp`

	userPrompt := fmt.Sprintf(`Dữ liệu đã phân tích từ các cuộc ghi âm:

%s

Câu hỏi: %s

Hãy trả lời câu hỏi dựa trên dữ liệu trên. Nếu không có thông tin, hãy nói "Không tìm thấy thông tin trong dữ liệu đã ghi".`, contextText, question)

	// Create OpenAI client
	client := openai.NewClient(apiKey)

	// Call OpenAI API
	ctx := context.Background()
	log.Printf("Calling OpenAI API to answer question...")

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
		Temperature: 0.3, // Low temperature for factual answers
		MaxTokens:   500, // Limit response length
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Printf("OpenAI API error while answering: %v", err)
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI returned no choices")
	}

	answer := strings.TrimSpace(resp.Choices[0].Message.Content)
	log.Printf("OpenAI answer received (length: %d)", len(answer))
	log.Printf("Usage - Prompt tokens: %d, Completion tokens: %d, Total tokens: %d",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	log.Printf("Answer: %s", answer)

	return answer, nil
}

// AnalysisContext represents analysis data with recording info
type AnalysisContext struct {
	RecordingID string
	CreatedAt   string
	Context     string
	Summary     []string
	ActionItems []string
	KeyPoints   []string
	Transcript  string
}

// buildContextFromAnalyses builds context text from all analyses
func buildContextFromAnalyses(analyses []AnalysisContext) string {
	if len(analyses) == 0 {
		return "Không có dữ liệu."
	}

	var builder strings.Builder
	builder.WriteString("Dữ liệu từ các cuộc ghi âm:\n\n")

	for i, analysis := range analyses {
		builder.WriteString(fmt.Sprintf("=== Ghi âm %d (ID: %s, %s) ===\n", i+1, analysis.RecordingID, analysis.CreatedAt))
		builder.WriteString(fmt.Sprintf("Loại: %s\n", analysis.Context))

		if len(analysis.Summary) > 0 {
			builder.WriteString("Tóm tắt:\n")
			for _, item := range analysis.Summary {
				builder.WriteString(fmt.Sprintf("- %s\n", item))
			}
		}

		if len(analysis.ActionItems) > 0 {
			builder.WriteString("Action Items:\n")
			for _, item := range analysis.ActionItems {
				builder.WriteString(fmt.Sprintf("- %s\n", item))
			}
		}

		if len(analysis.KeyPoints) > 0 {
			builder.WriteString("Điểm quan trọng:\n")
			for _, item := range analysis.KeyPoints {
				builder.WriteString(fmt.Sprintf("- %s\n", item))
			}
		}

		// Include transcript if available (truncated if too long)
		if analysis.Transcript != "" {
			transcript := analysis.Transcript
			if len(transcript) > 500 {
				transcript = transcript[:500] + "..."
			}
			builder.WriteString(fmt.Sprintf("Transcript: %s\n", transcript))
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

