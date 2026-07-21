package ai

import (
	"strings"
	"testing"
)

func TestFormatAvailableTagsPrompt(t *testing.T) {
	empty := formatAvailableTagsPrompt(nil)
	if !strings.Contains(empty, "none are defined") {
		t.Fatalf("expected empty-tags guidance, got %q", empty)
	}

	withTags := formatAvailableTagsPrompt([]string{" invoice ", "plumbing", "Invoice", ""})
	if !strings.Contains(withTags, "invoice") || !strings.Contains(withTags, "plumbing") {
		t.Fatalf("expected tag names in prompt, got %q", withTags)
	}
	if strings.Count(strings.ToLower(withTags), "invoice") != 1 {
		t.Fatalf("expected deduped invoice tag, got %q", withTags)
	}
	if !strings.Contains(withTags, "tags filter") {
		t.Fatalf("expected tags filter guidance, got %q", withTags)
	}
}

func TestBuildSearchSystemPromptIncludesTags(t *testing.T) {
	prompt := buildSearchSystemPrompt("en,de", "en", SearchModeDeep, []string{"invoice", "tax"})
	if !strings.Contains(prompt, "invoice") || !strings.Contains(prompt, "tax") {
		t.Fatalf("expected available tags in system prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "tags filter") {
		t.Fatalf("expected tags filter instruction, got %q", prompt)
	}
	if !strings.Contains(prompt, "deep search mode") {
		t.Fatalf("expected deep mode instruction, got %q", prompt)
	}
}
