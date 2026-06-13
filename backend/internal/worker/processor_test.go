package worker

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/models"
)

func newTestDocumentRecord() *core.Record {
	documents := core.NewBaseCollection("documents")
	documents.Fields.Add(
		&core.TextField{Name: "title"},
		&core.TextField{Name: "title_original"},
		&core.TextField{Name: "summary"},
		&core.TextField{Name: "summary_original"},
		&core.TextField{Name: "purpose"},
		&core.TextField{Name: "purpose_original"},
	)
	return core.NewRecord(documents)
}

func TestApplyExtractedMetadataWithoutTranslation(t *testing.T) {
	document := newTestDocumentRecord()
	metadata := &models.ExtractedMetadata{
		Title:   "Rechnung 001",
		Summary: "Eine Rechnung für Büromaterial.",
		Purpose: "Archivierung",
	}

	applyExtractedMetadata(document, metadata, "")

	if got := document.GetString("title"); got != "Rechnung 001" {
		t.Fatalf("expected title Rechnung 001, got %q", got)
	}
	if got := document.GetString("title_original"); got != "" {
		t.Fatalf("expected empty title_original, got %q", got)
	}
}

func TestApplyExtractedMetadataWithTranslation(t *testing.T) {
	document := newTestDocumentRecord()
	metadata := &models.ExtractedMetadata{
		Title:             "Rechnung 001",
		TitleTranslated:   "Invoice 001",
		Summary:           "Eine Rechnung für Büromaterial.",
		SummaryTranslated: "An invoice for office supplies.",
		Purpose:           "Archivierung",
		PurposeTranslated: "Archiving",
	}

	applyExtractedMetadata(document, metadata, "en")

	if got := document.GetString("title"); got != "Invoice 001" {
		t.Fatalf("expected title Invoice 001, got %q", got)
	}
	if got := document.GetString("title_original"); got != "Rechnung 001" {
		t.Fatalf("expected title_original Rechnung 001, got %q", got)
	}
}

func TestMergeTagNames(t *testing.T) {
	got := mergeTagNames([]string{"Rechnung", "Büro"}, []string{"Invoice", "Office"})
	if len(got) != 4 {
		t.Fatalf("expected 4 unique tags, got %d: %v", len(got), got)
	}
}
