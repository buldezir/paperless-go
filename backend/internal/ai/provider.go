package ai

import (
	"context"

	"paperless-go/backend/internal/models"
)

type Extractor interface {
	Name() string
	ExtractMetadata(ctx context.Context, ocrText string) (*models.ExtractedMetadata, error)
}
