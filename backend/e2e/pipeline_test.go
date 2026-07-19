package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestPipelineCompletesWithMocks(t *testing.T) {
	h := StartShared(t)
	token := h.userToken(t)

	rec := h.uploadDocument(t, token, h.UserID, fixturePath("sample.png"))
	id := jsonGetString(rec, "id")

	doc := h.waitDocumentStatus(t, token, id, "completed", "needs_review")
	ocr := jsonGetString(doc, "ocr_text")
	if !strings.Contains(ocr, "Acme Plumbing") {
		t.Fatalf("expected OCR text, got %q", ocr)
	}
	title := jsonGetString(doc, "title")
	if title == "" {
		t.Fatal("expected extracted title")
	}
	if !strings.Contains(strings.ToLower(title), "acme") && !strings.Contains(strings.ToLower(title), "invoice") {
		t.Fatalf("unexpected title %q", title)
	}

	// Job should exist and be completed.
	status, raw := h.doJSON(t, http.MethodGet, `/api/collections/processing_jobs/records?filter=document="`+id+`"&sort=-created&perPage=1`, token, nil)
	requireStatus(t, status, http.StatusOK, raw)
	jobs := mustDecodeList(t, raw)
	if len(jobs) == 0 {
		t.Fatal("expected processing job")
	}
	jobStatus := jsonGetString(jobs[0], "status")
	if jobStatus != "completed" && jobStatus != "needs_review" {
		t.Fatalf("job status=%q body=%s", jobStatus, raw)
	}
}

func TestPipelineReprocess(t *testing.T) {
	h := StartShared(t)
	token := h.userToken(t)

	rec := h.uploadDocument(t, token, h.UserID, fixturePath("sample.png"))
	id := jsonGetString(rec, "id")
	_ = h.waitDocumentStatus(t, token, id, "completed", "needs_review")

	status, raw := h.doJSON(t, http.MethodPost, "/api/collections/processing_jobs/records", token, map[string]any{
		"document":    id,
		"status":      "pending",
		"steps":       []string{"extract_metadata", "apply_metadata"},
		"force_steps": []string{"extract_metadata", "apply_metadata"},
	})
	requireStatus(t, status, http.StatusOK, raw)

	var job map[string]any
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		t.Fatalf("decode job: %v", err)
	}
	jobID := jsonGetString(job, "id")

	deadlineOK := false
	for i := 0; i < 100; i++ {
		status, raw = h.doJSON(t, http.MethodGet, "/api/collections/processing_jobs/records/"+jobID, token, nil)
		requireStatus(t, status, http.StatusOK, raw)
		_ = json.Unmarshal([]byte(raw), &job)
		st := jsonGetString(job, "status")
		if st == "completed" || st == "needs_review" || st == "failed" {
			deadlineOK = true
			if st == "failed" {
				t.Fatalf("reprocess failed: %s", raw)
			}
			break
		}
		h.waitDocumentStatus(t, token, id, "completed", "needs_review", "processing", "pending", "failed")
	}
	if !deadlineOK {
		t.Fatalf("reprocess job did not finish: %v", job)
	}
}
