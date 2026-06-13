package worker

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/ai"
	"paperless-go/backend/internal/config"
	"paperless-go/backend/internal/models"
	"paperless-go/backend/internal/ocr"
)

func Start(app core.App) {
	cfg := config.Load()
	ocrProvider := ocr.NewProvider(cfg.OCRProvider, cfg.OCRAPIKey)
	aiExtractor := ai.NewExtractor(
		cfg.OpenCodeGoAPIKey,
		cfg.OpenCodeGoModel,
		cfg.OpenCodeGoBaseURL,
		cfg.ExtractionPromptVer,
		cfg.OCRResultLanguage,
		cfg.OpenCodeGoTimeout,
	)

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		go runWorker(app, cfg, ocrProvider, aiExtractor)
		return e.Next()
	})
}

func runWorker(app core.App, cfg config.Config, ocrProvider ocr.Provider, aiExtractor ai.Extractor) {
	ticker := time.NewTicker(cfg.WorkerPollInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := processNextJob(app, cfg, ocrProvider, aiExtractor); err != nil {
			log.Printf("worker error: %v", err)
		}
	}
}

func processNextJob(app core.App, cfg config.Config, ocrProvider ocr.Provider, aiExtractor ai.Extractor) error {
	jobs, err := app.FindRecordsByFilter(
		"processing_jobs",
		"status = {:status}",
		"created",
		1,
		0,
		map[string]any{"status": models.JobStatusPending},
	)
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}

	job := jobs[0]
	return app.RunInTransaction(func(txApp core.App) error {
		return handleJob(txApp, cfg, job, ocrProvider, aiExtractor)
	})
}

func handleJob(app core.App, cfg config.Config, job *core.Record, ocrProvider ocr.Provider, aiExtractor ai.Extractor) error {
	job.Set("status", models.JobStatusRunning)
	job.Set("started_at", time.Now().Format("2006-01-02 15:04:05.000Z"))
	job.Set("ocr_provider", ocrProvider.Name())
	job.Set("ai_provider", aiExtractor.Name())
	job.Set("prompt_version", cfg.ExtractionPromptVer)
	if err := app.Save(job); err != nil {
		return err
	}

	document, err := app.FindRecordById("documents", job.GetString("document"))
	if err != nil {
		return failJob(app, job, nil, fmt.Errorf("load document: %w", err))
	}

	document.Set("processing_status", models.DocStatusProcessing)
	if err := app.Save(document); err != nil {
		return failJob(app, job, document, fmt.Errorf("mark document processing: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.OpenCodeGoTimeout+30*time.Second)
	defer cancel()

	ocrText, mimeType, err := extractOCRText(ctx, app, document, ocrProvider)
	if err != nil {
		return failJob(app, job, document, fmt.Errorf("ocr: %w", err))
	}

	metadata, err := aiExtractor.ExtractMetadata(ctx, ocrText)
	if err != nil {
		retryCount := int(job.GetFloat("retry_count"))
		if retryCount < cfg.WorkerMaxRetries {
			job.Set("retry_count", retryCount+1)
			job.Set("status", models.JobStatusPending)
			job.Set("error_message", err.Error())
			document.Set("processing_status", models.DocStatusPending)
			if saveErr := app.Save(document); saveErr != nil {
				return saveErr
			}
			return app.Save(job)
		}
		return failJob(app, job, document, fmt.Errorf("ai extraction: %w", err))
	}

	document.Set("ocr_text", ocrText)
	applyExtractedMetadata(document, metadata, cfg.OCRResultLanguage)
	if err := applyDocumentType(app, document, metadata, cfg.OCRResultLanguage); err != nil {
		return failJob(app, job, document, fmt.Errorf("document type: %w", err))
	}
	document.Set("confidence", metadata.Confidence)
	document.Set("people_or_organizations", metadata.PeopleOrOrganizations)
	document.Set("metadata_source", aiExtractor.Model())

	if metadata.DocumentDate != "" {
		document.Set("document_date", metadata.DocumentDate)
	}

	tagIDs, err := ensureTags(app, mergeTagNames(metadata.Tags, metadata.TagsTranslated))
	if err != nil {
		return failJob(app, job, document, fmt.Errorf("tags: %w", err))
	}
	document.Set("tags", tagIDs)

	status := models.DocStatusCompleted
	jobStatus := models.JobStatusCompleted
	if metadata.Confidence < 0.5 {
		status = models.DocStatusNeedsReview
		jobStatus = models.JobStatusNeedsReview
	}

	document.Set("processing_status", status)
	if err := app.Save(document); err != nil {
		return failJob(app, job, document, fmt.Errorf("save document: %w", err))
	}

	job.Set("status", jobStatus)
	job.Set("finished_at", time.Now().Format("2006-01-02 15:04:05.000Z"))
	job.Set("error_message", "")
	_ = mimeType

	return app.Save(job)
}

func extractOCRText(ctx context.Context, app core.App, document *core.Record, provider ocr.Provider) (string, string, error) {
	fileName := document.GetString("file")
	if fileName == "" {
		return "", "", fmt.Errorf("document has no file")
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", "", err
	}
	defer fsys.Close()

	fileKey := document.BaseFilesPath() + "/" + fileName
	reader, err := fsys.GetReader(fileKey)
	if err != nil {
		return "", "", fmt.Errorf("open uploaded file: %w", err)
	}
	defer reader.Close()

	tmpFile, err := os.CreateTemp("", "paperless-ocr-*"+filepath.Ext(fileName))
	if err != nil {
		return "", "", err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return "", "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", "", err
	}

	mimeType := guessMimeType(fileName)
	text, err := provider.ExtractText(ctx, tmpPath, mimeType)
	if err != nil {
		return "", mimeType, err
	}

	return text, mimeType, nil
}

func applyExtractedMetadata(document *core.Record, metadata *models.ExtractedMetadata, resultLanguage string) {
	if resultLanguage != "" {
		document.Set("title_original", metadata.Title)
		document.Set("summary_original", metadata.Summary)
		document.Set("purpose_original", metadata.Purpose)
		document.Set("title", firstNonEmpty(metadata.TitleTranslated, metadata.Title))
		document.Set("summary", firstNonEmpty(metadata.SummaryTranslated, metadata.Summary))
		document.Set("purpose", firstNonEmpty(metadata.PurposeTranslated, metadata.Purpose))
		return
	}

	document.Set("title", metadata.Title)
	document.Set("summary", metadata.Summary)
	document.Set("purpose", metadata.Purpose)
}

func mergeTagNames(original, translated []string) []string {
	seen := make(map[string]struct{}, len(original)+len(translated))
	names := make([]string, 0, len(original)+len(translated))

	for _, group := range [][]string{original, translated} {
		for _, rawName := range group {
			name := strings.TrimSpace(rawName)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			names = append(names, name)
		}
	}

	return names
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func documentTypeNames(metadata *models.ExtractedMetadata, resultLanguage string) (primary string, ensure []string) {
	original := strings.TrimSpace(metadata.DocumentType)
	if original == "" {
		return "", nil
	}
	if resultLanguage == "" {
		return original, []string{original}
	}

	translated := strings.TrimSpace(metadata.DocumentTypeTranslated)
	primary = firstNonEmpty(translated, original)
	return primary, mergeTagNames([]string{original}, []string{translated})
}

func applyDocumentType(app core.App, document *core.Record, metadata *models.ExtractedMetadata, resultLanguage string) error {
	primaryName, names := documentTypeNames(metadata, resultLanguage)
	if primaryName == "" {
		document.Set("document_type", "")
		return nil
	}

	typeID, err := ensureDocumentType(app, primaryName, names)
	if err != nil {
		return err
	}
	document.Set("document_type", typeID)
	return nil
}

func ensureDocumentType(app core.App, primaryName string, names []string) (string, error) {
	primaryName = strings.TrimSpace(primaryName)
	if primaryName == "" {
		return "", nil
	}

	collection, err := app.FindCollectionByNameOrId("document_types")
	if err != nil {
		return "", err
	}

	var primaryID string
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}

		id, err := findOrCreateDocumentType(app, collection, name)
		if err != nil {
			return "", err
		}
		if strings.EqualFold(name, primaryName) {
			primaryID = id
		}
	}

	if primaryID == "" {
		return findOrCreateDocumentType(app, collection, primaryName)
	}
	return primaryID, nil
}

func findDocumentTypeID(app core.App, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}

	existing, err := app.FindRecordsByFilter(
		"document_types",
		"name = {:name}",
		"",
		1,
		0,
		map[string]any{"name": name},
	)
	if err != nil {
		return "", err
	}
	if len(existing) == 0 {
		return "", nil
	}
	return existing[0].Id, nil
}

func findOrCreateDocumentType(app core.App, collection *core.Collection, name string) (string, error) {
	id, err := findDocumentTypeID(app, name)
	if err != nil {
		return "", err
	}
	if id != "" {
		return id, nil
	}

	record := core.NewRecord(collection)
	record.Set("name", name)
	if err := app.Save(record); err != nil {
		return "", err
	}
	return record.Id, nil
}

func ensureTags(app core.App, names []string) ([]string, error) {
	tagIDs := make([]string, 0, len(names))
	tagsCollection, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		return nil, err
	}

	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}

		existing, err := app.FindRecordsByFilter(
			"tags",
			"name = {:name}",
			"",
			1,
			0,
			map[string]any{"name": name},
		)
		if err != nil {
			return nil, err
		}

		if len(existing) > 0 {
			tagIDs = append(tagIDs, existing[0].Id)
			continue
		}

		tag := core.NewRecord(tagsCollection)
		tag.Set("name", name)
		if err := app.Save(tag); err != nil {
			return nil, err
		}
		tagIDs = append(tagIDs, tag.Id)
	}

	return tagIDs, nil
}

func failJob(app core.App, job *core.Record, document *core.Record, err error) error {
	job.Set("status", models.JobStatusFailed)
	job.Set("finished_at", time.Now().Format("2006-01-02 15:04:05.000Z"))
	job.Set("error_message", truncateError(err.Error(), 1900))
	if saveErr := app.Save(job); saveErr != nil {
		return saveErr
	}

	if document != nil {
		document.Set("processing_status", models.DocStatusFailed)
		if saveErr := app.Save(document); saveErr != nil {
			return saveErr
		}
	}

	return err
}

func guessMimeType(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func truncateError(msg string, max int) string {
	if len(msg) <= max {
		return msg
	}
	return msg[:max]
}
