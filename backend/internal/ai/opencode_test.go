package ai_test

import (
	"context"
	"testing"

	"paperless-go/backend/internal/ai"
)

func TestMockExtractorReturnsMetadata(t *testing.T) {
	extractor := ai.NewMockExtractor("v1")
	metadata, err := extractor.ExtractMetadata(context.Background(), "Invoice from Acme Supplies")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.Title != "Invoice" {
		t.Fatalf("expected title Invoice, got %q", metadata.Title)
	}
	if metadata.Confidence <= 0 {
		t.Fatal("expected positive confidence")
	}
}
