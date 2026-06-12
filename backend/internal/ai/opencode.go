package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"paperless-go/backend/internal/models"
)

const extractionSystemPrompt = `You extract structured metadata from OCR document text.
Return ONLY valid JSON with these fields:
- title (string, required)
- purpose (string)
- document_date (string, YYYY-MM-DD or empty)
- document_type (string)
- tags (array of strings)
- people_or_organizations (array of strings)
- summary (string, 1-3 sentences)
- confidence (number between 0 and 1)

Do not include markdown or explanation.`

type OpenCodeGoExtractor struct {
	apiKey       string
	model        string
	baseURL      string
	promptVer    string
	httpClient   *http.Client
}

func NewOpenCodeGoExtractor(apiKey, model, baseURL, promptVer string, timeout time.Duration) *OpenCodeGoExtractor {
	return &OpenCodeGoExtractor{
		apiKey:    apiKey,
		model:     model,
		baseURL:   strings.TrimRight(baseURL, "/"),
		promptVer: promptVer,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (e *OpenCodeGoExtractor) Name() string {
	return "opencode_go"
}

func (e *OpenCodeGoExtractor) ExtractMetadata(ctx context.Context, ocrText string) (*models.ExtractedMetadata, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("OPENCODE_GO_API_KEY is not configured")
	}

	reqBody := map[string]any{
		"model": e.model,
		"messages": []map[string]string{
			{"role": "system", "content": extractionSystemPrompt},
			{"role": "user", "content": fmt.Sprintf("Extract metadata from this OCR text:\n\n%s", truncate(ocrText, 12000))},
		},
		"response_format": map[string]string{
			"type": "json_object",
		},
		"temperature": 0.1,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := e.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("opencode go request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("opencode go error (%d): %s", resp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("decode opencode go response: %w", err)
	}
	if chatResp.Error != nil {
		return nil, fmt.Errorf("opencode go: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("opencode go returned no choices")
	}

	return models.ParseExtractedMetadata(chatResp.Choices[0].Message.Content)
}

func (e *OpenCodeGoExtractor) PromptVersion() string {
	return e.promptVer
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

type MockExtractor struct {
	promptVer string
}

func NewMockExtractor(promptVer string) *MockExtractor {
	return &MockExtractor{promptVer: promptVer}
}

func (m *MockExtractor) Name() string {
	return "mock"
}

func (m *MockExtractor) ExtractMetadata(ctx context.Context, ocrText string) (*models.ExtractedMetadata, error) {
	_ = ctx

	title := "Untitled Document"
	if strings.Contains(strings.ToLower(ocrText), "invoice") {
		title = "Invoice"
	}

	return &models.ExtractedMetadata{
		Title:                 title,
		Purpose:               "Document storage and review",
		DocumentDate:          "2024-03-15",
		DocumentType:          "invoice",
		Tags:                  []string{"invoice", "mock"},
		PeopleOrOrganizations: []string{"Acme Supplies Ltd."},
		Summary:               "Mock AI extraction for local development without OpenCode Go API key.",
		Confidence:            0.75,
	}, nil
}

func (m *MockExtractor) PromptVersion() string {
	return m.promptVer
}

func NewExtractor(apiKey, model, baseURL, promptVer string, timeout time.Duration) Extractor {
	if apiKey != "" {
		return NewOpenCodeGoExtractor(apiKey, model, baseURL, promptVer, timeout)
	}
	return NewMockExtractor(promptVer)
}
