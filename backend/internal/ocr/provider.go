package ocr

import (
	"context"
	"fmt"
	"log"
)

type Provider interface {
	Name() string
	ExtractText(ctx context.Context, filePath string, mimeType string) (string, error)
}

func NewProvider(name, apiKey string) (Provider, error) {
	switch name {
	case "google_vision":
		if apiKey == "" {
			return nil, fmt.Errorf("OCR_API_KEY is required when OCR_PROVIDER=google_vision")
		}
		log.Printf("[ocr] using provider=google_vision")
		return NewGoogleVisionProvider(apiKey), nil
	default:
		return nil, fmt.Errorf("unsupported OCR provider %q (supported: google_vision)", name)
	}
}
