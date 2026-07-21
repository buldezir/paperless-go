package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
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
			h.Mocks.ResetOpenAIBodies()
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

			bodies := h.Mocks.LastOpenAIBodies()
			if len(bodies) == 0 {
				t.Fatal("expected OpenAI search agent calls")
			}
			first := bodies[0]
			if !strings.Contains(first, "Available archive tags") {
				t.Fatalf("expected available tags in system prompt: %s", first)
			}
			if !strings.Contains(first, "plumbing") || !strings.Contains(first, "invoice") {
				t.Fatalf("expected archive tag names in first search interaction: %s", first)
			}
			if !strings.Contains(first, `"tags"`) {
				t.Fatalf("expected tags tool parameter in search_documents schema: %s", first)
			}
		})
	}
}

func TestAppDeepSearchAsSuperuserFindsUserDocuments(t *testing.T) {
	h := StartShared(t)
	userToken := h.userToken(t)
	superToken := h.superToken(t)

	rec := h.uploadDocument(t, userToken, h.UserID, fixturePath("sample.png"))
	id := jsonGetString(rec, "id")
	_ = h.waitDocumentStatus(t, userToken, id, "completed", "needs_review")

	// Homepage list as superuser sees all docs (rules bypass). Deep search must too.
	status, raw := h.doJSON(t, http.MethodGet, "/api/collections/documents/records?perPage=50", superToken, nil)
	requireStatus(t, status, http.StatusOK, raw)

	status, raw = h.doJSON(t, http.MethodPost, "/api/app/search", superToken, map[string]any{
		"mode": "shallow",
		"messages": []map[string]string{
			{"role": "user", "content": "Find the plumbing invoice about the leak"},
		},
	})
	requireStatus(t, status, http.StatusOK, raw)

	var out struct {
		Documents []map[string]any `json:"documents"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode: %v body %s", err, raw)
	}
	found := false
	for _, d := range out.Documents {
		if jsonGetString(d, "id") == id {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("superuser deep search should find user-owned docs like homepage search; got %s", raw)
	}
}
