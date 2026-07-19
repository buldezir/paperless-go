package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAppDeepSearchShallowAndDeep(t *testing.T) {
	h := StartShared(t)
	token := h.userToken(t)

	rec := h.uploadDocument(t, token, h.UserID, fixturePath("sample.png"))
	id := jsonGetString(rec, "id")
	_ = h.waitDocumentStatus(t, token, id, "completed", "needs_review")

	for _, mode := range []string{"shallow", "deep"} {
		t.Run(mode, func(t *testing.T) {
			status, raw := h.doJSON(t, http.MethodPost, "/api/app/search", token, map[string]any{
				"mode": mode,
				"messages": []map[string]string{
					{"role": "user", "content": "Find the plumbing invoice about the leak"},
				},
			})
			requireStatus(t, status, http.StatusOK, raw)

			var out struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Documents []map[string]any `json:"documents"`
			}
			if err := json.Unmarshal([]byte(raw), &out); err != nil {
				t.Fatalf("decode: %v body %s", err, raw)
			}
			if out.Message.Content == "" {
				t.Fatalf("empty search reply: %s", raw)
			}
			if len(out.Documents) == 0 {
				t.Fatalf("expected document hits: %s", raw)
			}
			found := false
			for _, d := range out.Documents {
				if jsonGetString(d, "id") == id {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("uploaded doc not in hits: %s", raw)
			}
		})
	}
}
