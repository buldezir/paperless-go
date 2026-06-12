package ocr

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) ExtractText(ctx context.Context, filePath string, mimeType string) (string, error) {
	_ = ctx

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file for mock OCR: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content != "" && (mimeType == "text/plain" || strings.HasSuffix(strings.ToLower(filePath), ".txt")) {
		return content, nil
	}

	base := filePath
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}

	return fmt.Sprintf(`Document: %s
Type: %s

This is mock OCR output for local development.
Upload a plain text file to get real extracted text, or configure a cloud OCR provider via OCR_PROVIDER and OCR_API_KEY.

Sample invoice content for AI extraction testing:
Invoice #INV-2024-001
Date: 2024-03-15
Vendor: Acme Supplies Ltd.
Purpose: Office equipment purchase
Total: $1,250.00
Tags: invoice, office, equipment
`, base, mimeType), nil
}
