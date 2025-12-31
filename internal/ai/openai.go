package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// AnalysisResult represents the AI analysis result
type AnalysisResult struct {
	Context      string   `json:"context"`
	Summary      []string `json:"summary"`
	ActionItems  []string `json:"action_items"`
	KeyPoints    []string `json:"key_points"`
	ZaloBrief    string   `json:"zalo_brief,omitempty"`
	Confidence   float64  `json:"confidence_score,omitempty"`
}

// AnalyzeTranscript analyzes transcript using OpenAI API
func AnalyzeTranscript(transcript string, detectedContext string) (*AnalysisResult, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	// Use rule-based context detection if not provided
	if detectedContext == "" {
		detectedContext = DetectContext(transcript)
	}

	// Build prompt (using simple version from day2.md)
	systemPrompt, userPrompt := BuildPrompt(transcript, detectedContext)
	
	log.Printf("=== OpenAI Analysis Request ===")
	log.Printf("Detected context: %s", detectedContext)
	log.Printf("Transcript length: %d characters", len(transcript))
	log.Printf("System prompt length: %d characters", len(systemPrompt))
	log.Printf("User prompt length: %d characters", len(userPrompt))

	// Create OpenAI client
	client := openai.NewClient(apiKey)

	// Call OpenAI API
	ctx := context.Background()
	log.Printf("Calling OpenAI API with model: GPT-4o-mini")
	
	req := openai.ChatCompletionRequest{
		Model: openai.GPT4oMini, // Using GPT-4o-mini as per MVP plan
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
		Temperature: 0.3, // Low temperature for factual output
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}
	
	resp, err := client.CreateChatCompletion(ctx, req)

	if err != nil {
		log.Printf("OpenAI API error: %v", err)
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	log.Printf("OpenAI API response received")
	log.Printf("Number of choices: %d", len(resp.Choices))
	log.Printf("Usage - Prompt tokens: %d, Completion tokens: %d, Total tokens: %d", 
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

	if len(resp.Choices) == 0 {
		log.Printf("ERROR: OpenAI returned no choices")
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	content := resp.Choices[0].Message.Content
	log.Printf("=== OpenAI Raw Response ===")
	log.Printf("Response length: %d characters", len(content))
	log.Printf("Response preview (first 500 chars): %s", truncateString(content, 500))
	log.Printf("Full response: %s", content)

	// Parse JSON response
	var result AnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("ERROR: Failed to parse OpenAI JSON directly. Error: %v", err)
		log.Printf("Attempting to extract JSON from markdown code blocks...")
		// Try to extract JSON from markdown code blocks
		extractedContent := extractJSONFromMarkdown(content)
		log.Printf("Extracted content: %s", extractedContent)
		if err := json.Unmarshal([]byte(extractedContent), &result); err != nil {
			log.Printf("ERROR: Failed to parse extracted JSON. Error: %v", err)
			return nil, fmt.Errorf("failed to parse OpenAI response as JSON: %w", err)
		}
		log.Printf("Successfully parsed JSON from markdown")
	} else {
		log.Printf("Successfully parsed JSON directly")
	}

	// Log parsed result
	log.Printf("=== Parsed Analysis Result ===")
	log.Printf("Context: %s", result.Context)
	log.Printf("Summary items: %d", len(result.Summary))
	log.Printf("Action items: %d", len(result.ActionItems))
	log.Printf("Key points: %d", len(result.KeyPoints))
	log.Printf("Zalo brief length: %d", len(result.ZaloBrief))
	
	if len(result.Summary) > 0 {
		log.Printf("Summary: %v", result.Summary)
	}
	if len(result.ActionItems) > 0 {
		log.Printf("Action items: %v", result.ActionItems)
	}
	if len(result.KeyPoints) > 0 {
		log.Printf("Key points: %v", result.KeyPoints)
	}
	if result.ZaloBrief != "" {
		log.Printf("Zalo brief: %s", result.ZaloBrief)
	}

	// Set context if not in response
	if result.Context == "" {
		log.Printf("Context missing in response, using detected context: %s", detectedContext)
		result.Context = detectedContext
	}

	// Generate zalo_brief from summary if missing
	if result.ZaloBrief == "" && len(result.Summary) > 0 {
		log.Printf("Zalo brief is empty, generating from summary...")
		result.ZaloBrief = generateZaloBrief(result.Summary)
		log.Printf("Generated zalo_brief: %s", result.ZaloBrief)
	}

	// Generate key_points from summary if missing
	if len(result.KeyPoints) == 0 && len(result.Summary) > 0 {
		log.Printf("Key points is empty, using summary as key points...")
		// Use first 3 summary items as key points
		maxPoints := 3
		if len(result.Summary) < maxPoints {
			maxPoints = len(result.Summary)
		}
		result.KeyPoints = result.Summary[:maxPoints]
		log.Printf("Generated key_points: %v", result.KeyPoints)
	}

	// Validate result
	if len(result.Summary) == 0 && len(result.ActionItems) == 0 && len(result.KeyPoints) == 0 {
		log.Printf("WARNING: Empty analysis result for transcript length: %d", len(transcript))
	}
	
	// Check for missing fields
	if len(result.Summary) == 0 {
		log.Printf("WARNING: Summary is empty")
	}
	if len(result.ActionItems) == 0 {
		log.Printf("INFO: Action items is empty (may be normal for thinking/lecture)")
	}
	if len(result.KeyPoints) == 0 {
		log.Printf("WARNING: Key points is still empty after fallback")
	}
	if result.ZaloBrief == "" {
		log.Printf("WARNING: Zalo brief is still empty after fallback")
	}

	log.Printf("=== Analysis Complete ===")
	return &result, nil
}

// generateZaloBrief generates zalo brief from summary
func generateZaloBrief(summary []string) string {
	if len(summary) == 0 {
		return ""
	}
	
	// Take first 3 items max
	maxItems := 3
	if len(summary) < maxItems {
		maxItems = len(summary)
	}
	
	brief := ""
	for i := 0; i < maxItems; i++ {
		brief += "- " + summary[i] + "\n"
	}
	
	return strings.TrimSpace(brief)
}

// extractJSONFromMarkdown extracts JSON from markdown code blocks
func extractJSONFromMarkdown(content string) string {
	// Remove markdown code blocks
	content = strings.TrimSpace(content)
	
	// Remove ```json and ```
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
	}
	
	return strings.TrimSpace(content)
}

// truncateString truncates string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

