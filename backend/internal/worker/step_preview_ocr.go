package worker

import (
	"context"
	"fmt"
	"strings"

	"paperless-go/backend/internal/models"
	"paperless-go/backend/internal/ocr"
	"paperless-go/backend/internal/preview"
)

type PreviewStep struct{}

func (s *PreviewStep) Name() string { return models.StepPreview }

func (s *PreviewStep) ShouldSkip(state *StepState) (bool, error) {
	if state.MimeType != "application/pdf" {
		return true, nil
	}
	if state.forced(models.StepPreview) {
		return false, nil
	}
	if state.Document.GetString("preview") != "" {
		return true, nil
	}
	return false, nil
}

func (s *PreviewStep) Run(ctx context.Context, state *StepState) error {
	_ = ctx
	if err := ensureTempFile(state); err != nil {
		return err
	}
	if state.MimeType != "application/pdf" {
		return nil
	}

	previewFile, err := preview.GenerateFirstPagePNG(state.TmpPath)
	if err != nil {
		return err
	}

	state.Document.Set("preview", previewFile)
	if err := state.App.Save(state.Document); err != nil {
		return fmt.Errorf("save preview: %w", err)
	}

	state.Logger.Info("preview saved", "file", previewFile.Name)
	return nil
}

type OCRStep struct {
	Provider ocr.Provider
}

func (s *OCRStep) Name() string { return models.StepOCR }

func (s *OCRStep) ShouldSkip(state *StepState) (bool, error) {
	if state.forced(models.StepOCR) {
		return false, nil
	}
	if strings.TrimSpace(state.Document.GetString("ocr_text")) != "" {
		state.OCRText = strings.TrimSpace(state.Document.GetString("ocr_text"))
		return true, nil
	}
	return false, nil
}

func (s *OCRStep) Run(ctx context.Context, state *StepState) error {
	if err := ensureTempFile(state); err != nil {
		return err
	}

	ocrCtx, cancel := context.WithTimeout(ctx, state.Cfg.OCRTimeout)
	defer cancel()

	ocrText, err := s.Provider.ExtractText(ocrCtx, state.TmpPath, state.MimeType)
	if err != nil {
		return fmt.Errorf("ocr: %w", err)
	}

	state.OCRText = ocrText
	state.Document.Set("ocr_text", ocrText)
	if err := state.App.Save(state.Document); err != nil {
		return fmt.Errorf("save ocr text: %w", err)
	}

	state.Logger.Info("OCR complete",
		"provider", s.Provider.Name(),
		"mime", state.MimeType,
		"chars", len(ocrText),
	)
	return nil
}

func ensureTempFile(state *StepState) error {
	if state.TmpPath != "" {
		return nil
	}

	tmpPath, mimeType, cleanup, err := readDocumentToTempFile(state.App, state.Document)
	if err != nil {
		return fmt.Errorf("read document file: %w", err)
	}
	state.TmpPath = tmpPath
	state.MimeType = mimeType
	if state.Cleanup == nil {
		state.Cleanup = cleanup
	}
	return nil
}
