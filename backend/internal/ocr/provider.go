package ocr

import "context"

type Provider interface {
	Name() string
	ExtractText(ctx context.Context, filePath string, mimeType string) (string, error)
}
