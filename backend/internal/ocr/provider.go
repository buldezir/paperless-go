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

type ProviderConfig struct {
	GoogleVisionAPIKey string
	MistralAPIKey      string
	MistralModel       string
	MistralBaseURL     string
}

func NewProvider(name string, cfg ProviderConfig) (Provider, error) {
	switch name {
	case "google_vision":
		if cfg.GoogleVisionAPIKey == "" {
			return nil, fmt.Errorf("GOOGLE_VISION_API_KEY is required when OCR_PROVIDER=google_vision")
		}
		log.Printf("[ocr] using provider=google_vision")
		return NewGoogleVisionProvider(cfg.GoogleVisionAPIKey), nil
	case "mistral":
		if cfg.MistralAPIKey == "" {
			return nil, fmt.Errorf("MISTRAL_API_KEY is required when OCR_PROVIDER=mistral")
		}
		log.Printf("[ocr] using provider=mistral model=%s", cfg.MistralModel)
		return NewMistralProvider(cfg.MistralAPIKey, cfg.MistralModel, cfg.MistralBaseURL), nil
	default:
		return nil, fmt.Errorf("unsupported OCR provider %q (supported: google_vision, mistral)", name)
	}
}
