package ai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Chatter interface {
	Chat(ctx context.Context, ocrText string, messages []ChatMessage) (string, error)
}

func buildChatSystemPrompt(ocrText string) string {
	return fmt.Sprintf(`You are a helpful assistant answering questions about a document.
Use the OCR text below as your primary source. If the answer is not in the document, say so clearly.
Be concise and accurate.

Document OCR text:

%s`, truncate(ocrText, 12000))
}

func (c *OpenAIClient) Chat(ctx context.Context, ocrText string, messages []ChatMessage) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not configured")
	}

	apiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	apiMessages = append(apiMessages, openai.SystemMessage(buildChatSystemPrompt(ocrText)))
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		content := strings.TrimSpace(msg.Content)
		if role == "" || content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			return "", fmt.Errorf("invalid message role: %s", role)
		}
		if role == "user" {
			apiMessages = append(apiMessages, openai.UserMessage(content))
		} else {
			apiMessages = append(apiMessages, openai.AssistantMessage(content))
		}
	}

	if len(apiMessages) < 2 {
		return "", fmt.Errorf("at least one user message is required")
	}

	requestStart := time.Now()
	log.Printf("[ai] chat completion model=%s base_url=%q messages=%d", c.model, c.baseURL, len(apiMessages))
	chatResp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       shared.ChatModel(c.model),
		Messages:    apiMessages,
		Temperature: openai.Float(0.3),
	})
	if err != nil {
		log.Printf("[ai] chat request failed duration=%s: %v", time.Since(requestStart).Round(time.Millisecond), err)
		return "", fmt.Errorf("openai chat completion: %w", err)
	}
	log.Printf("[ai] chat response choices=%d duration=%s",
		len(chatResp.Choices), time.Since(requestStart).Round(time.Millisecond))

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

func NewChatter(apiKey, model, baseURL string, timeout time.Duration) Chatter {
	return NewOpenAIClient(apiKey, model, baseURL, "", "", timeout)
}
