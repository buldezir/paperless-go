package ocr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	return &GoogleVisionProvider{
		client:  client,
		initErr: err,
	}
}

func (p *GoogleVisionProvider) Name() string {
	return "google_vision"
}

func (p *GoogleVisionProvider) ExtractText(ctx context.Context, filePath string, mimeType string) (string, error) {
	if p.initErr != nil {
		return "", fmt.Errorf("google vision client: %w", p.initErr)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file for OCR: %w", err)
	}

	if isVisionFileInput(mimeType, filePath) {
		return p.extractFileText(ctx, data, visionFileMimeType(mimeType, filePath))
	}

	return p.extractImageText(ctx, data, mimeType)
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

	for start := visionMaxFilePagesPerRequest + 1; start <= totalPages; start += visionMaxFilePagesPerRequest {
		end := start + visionMaxFilePagesPerRequest - 1
		if end > totalPages {
			end = totalPages
		}

		pages := make([]int32, 0, end-start+1)
		for page := start; page <= end; page++ {
			pages = append(pages, int32(page))
		}

		batch, err := p.annotateFile(ctx, content, mimeType, pages)
		if err != nil {
			return "", fmt.Errorf("ocr %s pages %d-%d: %w", mimeType, start, end, err)
		}
		parts = append(parts, batch.pageTexts...)
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

	return text, nil
}

func visionError(err error) error {
	return fmt.Errorf("google vision request: %w", err)
}

func NewProvider(name, apiKey string) Provider {
	switch name {
	case "google_vision":
		if apiKey != "" {
			return NewGoogleVisionProvider(apiKey)
		}
		fallthrough
	default:
		return NewMockProvider()
	}
}
