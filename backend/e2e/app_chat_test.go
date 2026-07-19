package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAppDocumentChat(t *testing.T) {
	h := StartShared(t)
	token := h.userToken(t)

	rec := h.uploadDocument(t, token, h.UserID, fixturePath("sample.png"))
	id := jsonGetString(rec, "id")
	doc := h.waitDocumentStatus(t, token, id, "completed", "needs_review")
	if strings.TrimSpace(jsonGetString(doc, "ocr_text")) == "" {
		t.Fatal("need ocr_text for chat")
	}

	status, raw := h.doJSON(t, http.MethodPost, "/api/app/documents/"+id+"/chat", token, map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": "What is the invoice total?"},
		},
	})
	requireStatus(t, status, http.StatusOK, raw)

	var out struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode: %v body %s", err, raw)
	}
	if out.Message.Content == "" {
		t.Fatalf("empty chat reply: %s", raw)
	}
	requireContains(t, out.Message.Content, "250")
}
