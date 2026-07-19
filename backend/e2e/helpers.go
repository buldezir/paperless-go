package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type authResult struct {
	Token  string
	Record map[string]any
}

func (h *Harness) authWithPassword(t testing.TB, collection, identity, password string) authResult {
	t.Helper()
	body := map[string]string{
		"identity": identity,
		"password": password,
	}
	var out struct {
		Token  string         `json:"token"`
		Record map[string]any `json:"record"`
	}
	status, raw := h.doJSON(t, http.MethodPost, "/api/collections/"+collection+"/auth-with-password", "", body)
	if status != http.StatusOK {
		t.Fatalf("auth %s as %s: status %d body %s", collection, identity, status, raw)
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode auth: %v body %s", err, raw)
	}
	if out.Token == "" {
		t.Fatal("empty auth token")
	}
	return authResult{Token: out.Token, Record: out.Record}
}

func (h *Harness) userToken(t testing.TB) string {
	t.Helper()
	return h.authWithPassword(t, "users", UserEmail, UserPassword).Token
}

func (h *Harness) superToken(t testing.TB) string {
	t.Helper()
	return h.authWithPassword(t, "_superusers", SuperEmail, SuperPassword).Token
}

func (h *Harness) doJSON(t testing.TB, method, path, token string, body any) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, h.BaseURL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := h.HTTP.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

func (h *Harness) doRaw(t testing.TB, method, path, token string, body io.Reader, contentType string) (int, string, http.Header) {
	t.Helper()
	req, err := http.NewRequest(method, h.BaseURL+path, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := h.HTTP.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw), resp.Header.Clone()
}

func (h *Harness) uploadDocument(t testing.TB, token, userID, filePath string) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("user", userID)
	_ = w.WriteField("processing_status", "pending")
	part, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	if _, err := io.Copy(part, f); err != nil {
		t.Fatalf("copy file: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	status, raw, _ := h.doRaw(t, http.MethodPost, "/api/collections/documents/records", token, &buf, w.FormDataContentType())
	if status < 200 || status >= 300 {
		t.Fatalf("upload document: status %d body %s", status, raw)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		t.Fatalf("decode document: %v body %s", err, raw)
	}
	return rec
}

func (h *Harness) getDocument(t testing.TB, token, id string) map[string]any {
	t.Helper()
	status, raw := h.doJSON(t, http.MethodGet, "/api/collections/documents/records/"+id, token, nil)
	if status != http.StatusOK {
		t.Fatalf("get document: status %d body %s", status, raw)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return rec
}

func (h *Harness) waitDocumentStatus(t testing.TB, token, id string, want ...string) map[string]any {
	t.Helper()
	wantSet := map[string]struct{}{}
	for _, s := range want {
		wantSet[s] = struct{}{}
	}
	deadline := time.Now().Add(60 * time.Second)
	var last map[string]any
	for time.Now().Before(deadline) {
		last = h.getDocument(t, token, id)
		status, _ := last["processing_status"].(string)
		if _, ok := wantSet[status]; ok {
			return last
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("document %s did not reach %v; last=%v", id, want, last)
	return nil
}

func fixturePath(name string) string {
	return filepath.Join("testdata", name)
}

func mustDecodeList(t testing.TB, raw string) []map[string]any {
	t.Helper()
	var out struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode list: %v body %s", err, raw)
	}
	return out.Items
}

func jsonGetString(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func requireContains(t testing.TB, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q in %q", needle, haystack)
	}
}

func requireStatus(t testing.TB, got, want int, body string) {
	t.Helper()
	if got != want {
		t.Fatalf("status %d want %d body %s", got, want, body)
	}
}

func formatErr(status int, body string) string {
	return fmt.Sprintf("status=%d body=%s", status, body)
}
