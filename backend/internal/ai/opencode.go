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

func buildExtractionSystemPrompt(resultLanguage string) string {
	prompt := `You extract structured metadata from OCR document text.
Return ONLY valid JSON with these fields:
- title (string, required)
- purpose (string)
- document_date (string, YYYY-MM-DD or empty)
- document_type (string)
- tags (array of strings)
- people_or_organizations (array of strings)
- summary (string, 1-3 sentences)
- confidence (number between 0 and 1)

Always write title, purpose, summary, tags, and people_or_organizations in the same language as the source document.`

	if resultLanguage != "" {
		prompt += fmt.Sprintf(`

Also include these fields translated into %s:
- title_translated (string)
- purpose_translated (string)
- summary_translated (string)
- tags_translated (array of strings) — one translation per tag, same order as tags`, resultLanguage)
	}

	prompt += `

Do not include markdown or explanation.`
	return prompt
}

type OpenCodeGoExtractor struct {
	apiKey         string
	model          string
	baseURL        string
	promptVer      string
	resultLanguage string
	httpClient     *http.Client
}

func NewOpenCodeGoExtractor(apiKey, model, baseURL, promptVer, resultLanguage string, timeout time.Duration) *OpenCodeGoExtractor {
	return &OpenCodeGoExtractor{
		apiKey:         apiKey,
		model:          model,
		baseURL:        strings.TrimRight(baseURL, "/"),
		promptVer:      promptVer,
		resultLanguage: resultLanguage,
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
			{"role": "system", "content": buildExtractionSystemPrompt(e.resultLanguage)},
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
	promptVer      string
	resultLanguage string
}

func NewMockExtractor(promptVer, resultLanguage string) *MockExtractor {
	return &MockExtractor{promptVer: promptVer, resultLanguage: resultLanguage}
}

func (m *MockExtractor) Name() string {
	return "mock"
}

func (m *MockExtractor) ExtractMetadata(ctx context.Context, ocrText string) (*models.ExtractedMetadata, error) {
	_ = ctx

	title := "Untitled Document"
	titleTranslated := ""
	if strings.Contains(strings.ToLower(ocrText), "invoice") {
		title = "Invoice"
		if m.resultLanguage == "de" {
			titleTranslated = "Rechnung"
		} else if m.resultLanguage == "en" && !strings.Contains(strings.ToLower(ocrText), "rechnung") {
			titleTranslated = title
		} else if m.resultLanguage == "en" {
			titleTranslated = "Invoice"
		}
	}

	summary := "Mock AI extraction for local development without OpenCode Go API key."
	summaryTranslated := ""
	if m.resultLanguage == "de" {
		summaryTranslated = "Mock-KI-Extraktion für die lokale Entwicklung ohne OpenCode-Go-API-Schlüssel."
	} else if m.resultLanguage == "en" {
		summaryTranslated = summary
	}

	tags := []string{"invoice", "mock"}
	tagsTranslated := []string(nil)
	if m.resultLanguage == "de" {
		tagsTranslated = []string{"Rechnung", "Mock"}
	} else if m.resultLanguage == "en" {
		tagsTranslated = tags
	}

	return &models.ExtractedMetadata{
		Title:                 title,
		TitleTranslated:       titleTranslated,
		Purpose:               "Document storage and review",
		PurposeTranslated:     mockPurposeTranslated(m.resultLanguage),
		DocumentDate:          "2024-03-15",
		DocumentType:          "invoice",
		Tags:                  tags,
		TagsTranslated:        tagsTranslated,
		PeopleOrOrganizations: []string{"Acme Supplies Ltd."},
		Summary:               summary,
		SummaryTranslated:     summaryTranslated,
		Confidence:            0.75,
	}, nil
}

func mockPurposeTranslated(resultLanguage string) string {
	switch resultLanguage {
	case "de":
		return "Dokumentenspeicherung und -prüfung"
	case "en":
		return "Document storage and review"
	default:
		return ""
	}
}

func (m *MockExtractor) PromptVersion() string {
	return m.promptVer
}

func NewExtractor(apiKey, model, baseURL, promptVer, resultLanguage string, timeout time.Duration) Extractor {
	if apiKey != "" {
		return NewOpenCodeGoExtractor(apiKey, model, baseURL, promptVer, resultLanguage, timeout)
	}
	return NewMockExtractor(promptVer, resultLanguage)
}
