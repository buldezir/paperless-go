package models_test

import (
	"testing"

	"paperless-go/backend/internal/models"
)

func TestParseExtractedMetadataValid(t *testing.T) {
	raw := `{
		"title": "Invoice 001",
		"purpose": "Office supplies",
		"document_date": "2024-03-15",
		"document_type": "invoice",
		"tags": ["invoice", "office"],
		"people_or_organizations": ["Acme Ltd."],
		"summary": "An invoice for office supplies.",
		"confidence": 0.92
	}`

	metadata, err := models.ParseExtractedMetadata(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metadata.Title != "Invoice 001" {
		t.Fatalf("expected title Invoice 001, got %q", metadata.Title)
	}
	if metadata.DocumentDate != "2024-03-15" {
		t.Fatalf("expected date 2024-03-15, got %q", metadata.DocumentDate)
	}
}

func TestParseExtractedMetadataInvalidJSON(t *testing.T) {
	_, err := models.ParseExtractedMetadata("{not-json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseExtractedMetadataMissingTitle(t *testing.T) {
	raw := `{"title":"","confidence":0.5}`
	_, err := models.ParseExtractedMetadata(raw)
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
}

func TestParseExtractedMetadataStripsMarkdownFence(t *testing.T) {
	raw := "```json\n{\"title\":\"Test\",\"confidence\":0.8}\n```"
	metadata, err := models.ParseExtractedMetadata(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metadata.Title != "Test" {
		t.Fatalf("expected title Test, got %q", metadata.Title)
	}
}

func TestParseExtractedMetadataStripsReasoningTags(t *testing.T) {
	raw := "<think>\nThe user wants JSON metadata.\n</think>\n{\"title\":\"Invoice 001\",\"confidence\":0.9}"
	metadata, err := models.ParseExtractedMetadata(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metadata.Title != "Invoice 001" {
		t.Fatalf("expected title Invoice 001, got %q", metadata.Title)
	}
}
