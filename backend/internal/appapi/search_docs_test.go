package appapi

import (
	"strings"
	"testing"
)

func TestOCRSnippet(t *testing.T) {
	ocr := "Preface text. The plumber invoice for the leak was paid in July. Trailing notes."
	got := ocrSnippet(ocr, "plumber invoice")
	if got == "" {
		t.Fatal("expected snippet")
	}
	if !strings.Contains(strings.ToLower(got), "plumber") {
		t.Fatalf("expected plumber in snippet, got %q", got)
	}
}

func TestTruncateRunes(t *testing.T) {
	got := truncateRunes("abcdefghij", 5)
	if got != "abcde…" {
		t.Fatalf("got %q", got)
	}
}
