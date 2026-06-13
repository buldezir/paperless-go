package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
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

func (e *OpenCodeGoExtractor) Chat(ctx context.Context, ocrText string, messages []ChatMessage) (string, error) {
	if e.apiKey == "" {
		return "", fmt.Errorf("OPENCODE_GO_API_KEY is not configured")
	}

	apiMessages := make([]map[string]string, 0, len(messages)+1)
	apiMessages = append(apiMessages, map[string]string{
		"role":    "system",
		"content": buildChatSystemPrompt(ocrText),
	})
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		content := strings.TrimSpace(msg.Content)
		if role == "" || content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			return "", fmt.Errorf("invalid message role: %s", role)
		}
		apiMessages = append(apiMessages, map[string]string{
			"role":    role,
			"content": content,
		})
	}

	if len(apiMessages) < 2 {
		return "", fmt.Errorf("at least one user message is required")
	}

	reqBody := map[string]any{
		"model":       e.model,
		"messages":    apiMessages,
		"temperature": 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := e.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	requestStart := time.Now()
	log.Printf("[ai] chat POST %s model=%s messages=%d request_bytes=%d", url, e.model, len(apiMessages), len(body))
	resp, err := e.httpClient.Do(req)
	if err != nil {
		log.Printf("[ai] chat request failed duration=%s: %v", time.Since(requestStart).Round(time.Millisecond), err)
		return "", fmt.Errorf("opencode go request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	log.Printf("[ai] chat response status=%d bytes=%d duration=%s",
		resp.StatusCode, len(respBody), time.Since(requestStart).Round(time.Millisecond))

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("opencode go error (%d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("decode opencode go response: %w", err)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("opencode go: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("opencode go returned no choices")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

func NewChatter(apiKey, model, baseURL string, timeout time.Duration) Chatter {
	return NewOpenCodeGoExtractor(apiKey, model, baseURL, "", "", timeout)
}
