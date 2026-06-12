package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type GoogleVisionProvider struct {
	apiKey     string
	httpClient *http.Client
}

func NewGoogleVisionProvider(apiKey string) *GoogleVisionProvider {
	return &GoogleVisionProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *GoogleVisionProvider) Name() string {
	return "google_vision"
}

func (p *GoogleVisionProvider) ExtractText(ctx context.Context, filePath string, mimeType string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file for OCR: %w", err)
	}

	payload := map[string]any{
		"requests": []map[string]any{
			{
				"image": map[string]string{
					"content": base64.StdEncoding.EncodeToString(data),
				},
				"features": []map[string]string{
					{"type": "DOCUMENT_TEXT_DETECTION"},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("google vision request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("google vision error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Responses []struct {
			FullTextAnnotation struct {
				Text string `json:"text"`
			} `json:"fullTextAnnotation"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"responses"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode google vision response: %w", err)
	}

	if len(result.Responses) == 0 {
		return "", fmt.Errorf("google vision returned no responses")
	}
	if result.Responses[0].Error.Message != "" {
		return "", fmt.Errorf("google vision: %s", result.Responses[0].Error.Message)
	}

	text := result.Responses[0].FullTextAnnotation.Text
	if text == "" {
		return "", fmt.Errorf("google vision returned empty text for mime type %s", mimeType)
	}

	return text, nil
}

func NewProvider(name, apiKey string) Provider {
	switch name {
	case "google_vision":
		if apiKey != "" {
			return NewGoogleVisionProvider(apiKey)
		}
		fallthrough
	default:
		return NewMockProvider()
	}
}
