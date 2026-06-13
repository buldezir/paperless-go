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

	"paperless-go/backend/internal/models"
)

func buildExtractionSystemPrompt(resultLanguage string) string {
	prompt := `You extract structured metadata from OCR document text.
Return ONLY valid JSON with these fields:
- title (string, required)
- purpose (string)
- document_date (string, YYYY-MM-DD or empty)
- document_type (string)
- correspondent (string, primary sender or issuer)
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
- document_type_translated (string)
- correspondent_translated (string)
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

func (e *OpenCodeGoExtractor) Model() string {
	return e.model
}

func (e *OpenCodeGoExtractor) ExtractMetadata(ctx context.Context, ocrText string) (*models.ExtractedMetadata, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("OPENCODE_GO_API_KEY is not configured")
	}

	inputChars := len(ocrText)
	sentChars := len(truncate(ocrText, 12000))
	log.Printf("[ai] extraction starting provider=%s model=%s prompt_ver=%s ocr_chars=%d sent_chars=%d result_lang=%q",
		e.Name(), e.model, e.promptVer, inputChars, sentChars, e.resultLanguage)

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

	requestStart := time.Now()
	log.Printf("[ai] POST %s request_bytes=%d", url, len(body))
	resp, err := e.httpClient.Do(req)
	if err != nil {
		log.Printf("[ai] request failed duration=%s: %v", time.Since(requestStart).Round(time.Millisecond), err)
		return nil, fmt.Errorf("opencode go request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Printf("[ai] response status=%d bytes=%d duration=%s",
		resp.StatusCode, len(respBody), time.Since(requestStart).Round(time.Millisecond))

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

	content := chatResp.Choices[0].Message.Content
	metadata, err := models.ParseExtractedMetadata(content)
	if err != nil {
		log.Printf("[ai] parse failed content_chars=%d: %v", len(content), err)
		return nil, err
	}
	log.Printf("[ai] extraction complete confidence=%.2f title=%q type=%q tags=%d content_chars=%d",
		metadata.Confidence, truncateForLog(metadata.Title, 80), truncateForLog(metadata.DocumentType, 40),
		len(metadata.Tags), len(content))
	return metadata, nil
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

func truncateForLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func NewExtractor(apiKey, model, baseURL, promptVer, resultLanguage string, timeout time.Duration) Extractor {
	return NewOpenCodeGoExtractor(apiKey, model, baseURL, promptVer, resultLanguage, timeout)
}
