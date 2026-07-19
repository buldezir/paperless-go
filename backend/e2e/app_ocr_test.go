package e2e

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
)

func TestAppOCRProviders(t *testing.T) {
	h := StartShared(t)
	token := h.userToken(t)
	status, raw := h.doJSON(t, http.MethodGet, "/api/app/ocr/providers", token, nil)
	requireStatus(t, status, http.StatusOK, raw)
	requireContains(t, raw, "mistral")
}

func TestAppOCRTest(t *testing.T) {
	h := StartShared(t)
	token := h.userToken(t)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("provider", "mistral")
	part, err := w.CreateFormFile("file", "sample.png")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(fixturePath("sample.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := io.Copy(part, f); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	status, raw, _ := h.doRaw(t, http.MethodPost, "/api/app/ocr/test", token, &buf, w.FormDataContentType())
	requireStatus(t, status, http.StatusOK, raw)
	requireContains(t, raw, "Acme Plumbing")
}
