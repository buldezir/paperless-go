package ocr

import (
	"context"
	"fmt"
	"log/slog"
	"time"
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
	OCRTimeout         time.Duration
	Logger             *slog.Logger
}

type ProviderInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func AvailableProviders(cfg ProviderConfig) []ProviderInfo {
	providers := make([]ProviderInfo, 0, 2)
	if cfg.GoogleVisionAPIKey != "" {
		providers = append(providers, ProviderInfo{ID: "google_vision", Name: "Google Cloud Vision"})
	}
	if cfg.MistralAPIKey != "" {
		providers = append(providers, ProviderInfo{ID: "mistral", Name: "Mistral OCR"})
	}
	return providers
}

func NewProvider(name string, cfg ProviderConfig) (Provider, error) {
	switch name {
	case "google_vision":
		if cfg.GoogleVisionAPIKey == "" {
			return nil, fmt.Errorf("GOOGLE_VISION_API_KEY is required when OCR_PROVIDER=google_vision")
		}
		cfg.Logger.Info("using provider", "provider", "google_vision")
		return NewGoogleVisionProvider(cfg.GoogleVisionAPIKey, cfg.Logger), nil
	case "mistral":
		if cfg.MistralAPIKey == "" {
			return nil, fmt.Errorf("MISTRAL_API_KEY is required when OCR_PROVIDER=mistral")
		}
		cfg.Logger.Info("using provider", "provider", "mistral", "model", cfg.MistralModel)
		return NewMistralProvider(cfg.MistralAPIKey, cfg.MistralModel, cfg.MistralBaseURL, cfg.OCRTimeout, cfg.Logger), nil
	default:
		return nil, fmt.Errorf("unsupported OCR provider %q (supported: google_vision, mistral)", name)
	}
}
