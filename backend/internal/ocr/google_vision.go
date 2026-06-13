package ocr

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	vision "cloud.google.com/go/vision/v2/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"google.golang.org/api/option"
)

const visionMaxFilePagesPerRequest = 5

func visionDocumentTextFeatures() []*visionpb.Feature {
	return []*visionpb.Feature{
		{Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION},
	}
}

type GoogleVisionProvider struct {
	client  *vision.ImageAnnotatorClient
	initErr error
}

func NewGoogleVisionProvider(apiKey string) *GoogleVisionProvider {
	client, err := vision.NewImageAnnotatorClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("[ocr] google vision client init failed: %v", err)
	} else {
		log.Printf("[ocr] google vision client initialized")
	}
	return &GoogleVisionProvider{
		client:  client,
		initErr: err,
	}
}

func (p *GoogleVisionProvider) Name() string {
	return "google_vision"
}

func (p *GoogleVisionProvider) ExtractText(ctx context.Context, filePath string, mimeType string) (string, error) {
	start := time.Now()
	if p.initErr != nil {
		return "", fmt.Errorf("google vision client: %w", p.initErr)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file for OCR: %w", err)
	}

	effectiveMime := mimeType
	mode := "image"
	if isVisionFileInput(mimeType, filePath) {
		effectiveMime = visionFileMimeType(mimeType, filePath)
		mode = "file"
	}
	log.Printf("[ocr] google vision starting file=%q mime=%s effective_mime=%s mode=%s bytes=%d",
		filepath.Base(filePath), mimeType, effectiveMime, mode, len(data))

	var text string
	if mode == "file" {
		text, err = p.extractFileText(ctx, data, effectiveMime)
	} else {
		text, err = p.extractImageText(ctx, data, mimeType)
	}
	if err != nil {
		log.Printf("[ocr] google vision failed file=%q duration=%s: %v",
			filepath.Base(filePath), time.Since(start).Round(time.Millisecond), err)
		return "", err
	}
	log.Printf("[ocr] google vision complete file=%q chars=%d duration=%s",
		filepath.Base(filePath), len(text), time.Since(start).Round(time.Millisecond))
	return text, nil
}

func isVisionFileInput(mimeType, filePath string) bool {
	switch mimeType {
	case "application/pdf", "image/tiff", "image/gif":
		return true
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".pdf", ".tif", ".tiff", ".gif":
		return true
	default:
		return false
	}
}

func visionFileMimeType(mimeType, filePath string) string {
	if mimeType != "" && mimeType != "application/octet-stream" {
		return mimeType
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".pdf":
		return "application/pdf"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".gif":
		return "image/gif"
	default:
		return mimeType
	}
}

func (p *GoogleVisionProvider) extractFileText(ctx context.Context, content []byte, mimeType string) (string, error) {
	first, err := p.annotateFile(ctx, content, mimeType, nil)
	if err != nil {
		return "", err
	}

	parts := append([]string{}, first.pageTexts...)
	totalPages := first.totalPages
	log.Printf("[ocr] google vision file mime=%s total_pages=%d first_batch_pages=%d",
		mimeType, totalPages, len(first.pageTexts))

	for start := visionMaxFilePagesPerRequest + 1; start <= totalPages; start += visionMaxFilePagesPerRequest {
		end := start + visionMaxFilePagesPerRequest - 1
		if end > totalPages {
			end = totalPages
		}

		pages := make([]int32, 0, end-start+1)
		for page := start; page <= end; page++ {
			pages = append(pages, int32(page))
		}

		log.Printf("[ocr] google vision file batch pages=%d-%d", start, end)
		batch, err := p.annotateFile(ctx, content, mimeType, pages)
		if err != nil {
			return "", fmt.Errorf("ocr %s pages %d-%d: %w", mimeType, start, end, err)
		}
		parts = append(parts, batch.pageTexts...)
		log.Printf("[ocr] google vision file batch pages=%d-%d extracted=%d", start, end, len(batch.pageTexts))
	}

	text := strings.Join(parts, "\n\n")
	if text == "" {
		return "", fmt.Errorf("google vision returned empty text for mime type %s", mimeType)
	}

	return text, nil
}

type fileAnnotateResult struct {
	pageTexts  []string
	totalPages int
}

func (p *GoogleVisionProvider) annotateFile(ctx context.Context, content []byte, mimeType string, pages []int32) (fileAnnotateResult, error) {
	start := time.Now()
	req := &visionpb.BatchAnnotateFilesRequest{
		Requests: []*visionpb.AnnotateFileRequest{
			{
				InputConfig: &visionpb.InputConfig{
					Content:  content,
					MimeType: mimeType,
				},
				Features: visionDocumentTextFeatures(),
				Pages:    pages,
			},
		},
	}

	resp, err := p.client.BatchAnnotateFiles(ctx, req)
	if err != nil {
		return fileAnnotateResult{}, visionError(err)
	}
	if len(resp.GetResponses()) == 0 {
		return fileAnnotateResult{}, fmt.Errorf("google vision returned no file responses")
	}

	fileResp := resp.GetResponses()[0]
	if fileResp.GetError() != nil {
		return fileAnnotateResult{}, fmt.Errorf("google vision: %s", fileResp.GetError().GetMessage())
	}

	pageTexts := make([]string, 0, len(fileResp.GetResponses()))
	for i, pageResp := range fileResp.GetResponses() {
		if pageResp.GetError() != nil {
			return fileAnnotateResult{}, fmt.Errorf("google vision page %d: %s", visionPageNumber(pages, i), pageResp.GetError().GetMessage())
		}
		if text := strings.TrimSpace(pageResp.GetFullTextAnnotation().GetText()); text != "" {
			pageTexts = append(pageTexts, text)
		}
	}

	log.Printf("[ocr] google vision BatchAnnotateFiles mime=%s pages=%v extracted=%d total_pages=%d duration=%s",
		mimeType, pages, len(pageTexts), fileResp.GetTotalPages(), time.Since(start).Round(time.Millisecond))
	return fileAnnotateResult{
		pageTexts:  pageTexts,
		totalPages: int(fileResp.GetTotalPages()),
	}, nil
}

func visionPageNumber(pages []int32, index int) int32 {
	if index < len(pages) {
		return pages[index]
	}
	return int32(index + 1)
}

func (p *GoogleVisionProvider) extractImageText(ctx context.Context, content []byte, mimeType string) (string, error) {
	start := time.Now()
	req := &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image:    &visionpb.Image{Content: content},
				Features: visionDocumentTextFeatures(),
			},
		},
	}

	resp, err := p.client.BatchAnnotateImages(ctx, req)
	if err != nil {
		return "", visionError(err)
	}
	if len(resp.GetResponses()) == 0 {
		return "", fmt.Errorf("google vision returned no responses")
	}

	imageResp := resp.GetResponses()[0]
	if imageResp.GetError() != nil {
		return "", fmt.Errorf("google vision: %s", imageResp.GetError().GetMessage())
	}

	text := imageResp.GetFullTextAnnotation().GetText()
	if text == "" {
		return "", fmt.Errorf("google vision returned empty text for mime type %s", mimeType)
	}

	log.Printf("[ocr] google vision BatchAnnotateImages mime=%s chars=%d duration=%s",
		mimeType, len(text), time.Since(start).Round(time.Millisecond))
	return text, nil
}

func visionError(err error) error {
	return fmt.Errorf("google vision request: %w", err)
}
