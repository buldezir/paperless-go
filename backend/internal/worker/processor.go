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
	"paperless-go/backend/internal/preview"
)

func Start(app core.App) {
	cfg := config.Load()
	ocrProvider, err := ocr.NewProvider(cfg.OCRProvider, cfg.OCRAPIKey)
	if err != nil {
		log.Fatalf("[worker] OCR provider: %v", err)
	}
	aiExtractor := ai.NewExtractor(
		cfg.OpenCodeGoAPIKey,
		cfg.OpenCodeGoModel,
		cfg.OpenCodeGoBaseURL,
		cfg.ExtractionPromptVer,
		cfg.OCRResultLanguage,
		cfg.OpenCodeGoTimeout,
	)

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		log.Printf("[worker] starting poll_interval=%s ocr=%s ai=%s model=%s",
			cfg.WorkerPollInterval, ocrProvider.Name(), aiExtractor.Name(), aiExtractor.Model())
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
	log.Printf("[worker] picked job=%s document=%s type=%s retry=%d",
		job.Id, job.GetString("document"), job.GetString("job_type"), int(job.GetFloat("retry_count")))
	return app.RunInTransaction(func(txApp core.App) error {
		return handleJob(txApp, cfg, job, ocrProvider, aiExtractor)
	})
}

func handleJob(app core.App, cfg config.Config, job *core.Record, ocrProvider ocr.Provider, aiExtractor ai.Extractor) error {
	jobStart := time.Now()
	documentID := job.GetString("document")
	jobType := job.GetString("job_type")
	log.Printf("[worker] job=%s document=%s type=%s starting", job.Id, documentID, jobType)

	job.Set("status", models.JobStatusRunning)
	job.Set("started_at", time.Now().Format("2006-01-02 15:04:05.000Z"))
	job.Set("ai_provider", aiExtractor.Name())
	job.Set("ai_model", aiExtractor.Model())
	job.Set("prompt_version", cfg.ExtractionPromptVer)
	if job.GetString("job_type") != models.JobTypeExtraction {
		job.Set("ocr_provider", ocrProvider.Name())
	}
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

	var ocrText string
	var mimeType string
	switch jobType {
	case models.JobTypeExtraction:
		ocrText = strings.TrimSpace(document.GetString("ocr_text"))
		if ocrText == "" {
			return failJob(app, job, document, fmt.Errorf("extraction reprocess requires existing ocr_text"))
		}
		log.Printf("[worker] job=%s document=%s skipping OCR, reusing stored text (%d chars)",
			job.Id, documentID, len(ocrText))
	default:
		tmpPath, mimeType, cleanup, err := readDocumentToTempFile(app, document)
		if err != nil {
			return failJob(app, job, document, fmt.Errorf("read document file: %w", err))
		}
		defer cleanup()

		if err := attachPDFPreview(app, document, tmpPath, mimeType); err != nil {
			log.Printf("[worker] job=%s document=%s preview failed: %v", job.Id, documentID, err)
		}

		ocrStart := time.Now()
		ocrText, err = ocrProvider.ExtractText(ctx, tmpPath, mimeType)
		if err != nil {
			return failJob(app, job, document, fmt.Errorf("ocr: %w", err))
		}
		log.Printf("[worker] job=%s document=%s OCR complete provider=%s mime=%s chars=%d duration=%s",
			job.Id, documentID, ocrProvider.Name(), mimeType, len(ocrText), time.Since(ocrStart).Round(time.Millisecond))
	}

	log.Printf("[worker] job=%s document=%s starting AI extraction provider=%s model=%s ocr_chars=%d",
		job.Id, documentID, aiExtractor.Name(), aiExtractor.Model(), len(ocrText))
	aiStart := time.Now()
	metadata, err := aiExtractor.ExtractMetadata(ctx, ocrText)
	if err != nil {
		log.Printf("[worker] job=%s document=%s AI extraction failed duration=%s: %v",
			job.Id, documentID, time.Since(aiStart).Round(time.Millisecond), err)
		retryCount := int(job.GetFloat("retry_count"))
		if retryCount < cfg.WorkerMaxRetries {
			log.Printf("[worker] job=%s document=%s scheduling retry %d/%d",
				job.Id, documentID, retryCount+1, cfg.WorkerMaxRetries)
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
	log.Printf("[worker] job=%s document=%s AI extraction complete duration=%s confidence=%.2f title=%q type=%q tags=%d",
		job.Id, documentID, time.Since(aiStart).Round(time.Millisecond), metadata.Confidence,
		truncateForLog(metadata.Title, 80), truncateForLog(metadata.DocumentType, 40), len(metadata.Tags))

	if jobType != models.JobTypeExtraction {
		document.Set("ocr_text", ocrText)
	}
	applyExtractedMetadata(document, metadata, cfg.OCRResultLanguage)
	if err := applyDocumentType(app, document, metadata, cfg.OCRResultLanguage); err != nil {
		return failJob(app, job, document, fmt.Errorf("document type: %w", err))
	}
	log.Printf("[worker] job=%s document=%s document_type=%s",
		job.Id, documentID, truncateForLog(document.GetString("document_type"), 40))
	if err := applyCorrespondent(app, document, metadata, cfg.OCRResultLanguage); err != nil {
		return failJob(app, job, document, fmt.Errorf("correspondent: %w", err))
	}
	log.Printf("[worker] job=%s document=%s correspondent=%s",
		job.Id, documentID, truncateForLog(document.GetString("correspondent"), 40))
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
	log.Printf("[worker] job=%s document=%s applied %d tags", job.Id, documentID, len(tagIDs))

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

	log.Printf("[worker] job=%s document=%s finished status=%s duration=%s",
		job.Id, documentID, jobStatus, time.Since(jobStart).Round(time.Millisecond))
	return app.Save(job)
}

func readDocumentToTempFile(app core.App, document *core.Record) (tmpPath, mimeType string, cleanup func(), err error) {
	fileName := document.GetString("file")
	if fileName == "" {
		return "", "", func() {}, fmt.Errorf("document has no file")
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", "", func() {}, err
	}
	defer fsys.Close()

	fileKey := document.BaseFilesPath() + "/" + fileName
	reader, err := fsys.GetReader(fileKey)
	if err != nil {
		return "", "", func() {}, fmt.Errorf("open uploaded file: %w", err)
	}
	defer reader.Close()

	tmpFile, err := os.CreateTemp("", "paperless-doc-*"+filepath.Ext(fileName))
	if err != nil {
		return "", "", func() {}, err
	}
	tmpPath = tmpFile.Name()
	cleanup = func() { os.Remove(tmpPath) }

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		cleanup()
		return "", "", func() {}, err
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", "", func() {}, err
	}

	return tmpPath, guessMimeType(fileName), cleanup, nil
}

func attachPDFPreview(app core.App, document *core.Record, filePath, mimeType string) error {
	if mimeType != "application/pdf" {
		return nil
	}

	previewFile, err := preview.GenerateFirstPagePNG(filePath)
	if err != nil {
		return err
	}

	document.Set("preview", previewFile)
	if err := app.Save(document); err != nil {
		return fmt.Errorf("save preview: %w", err)
	}

	log.Printf("[worker] document=%s preview saved file=%q", document.Id, previewFile.Name)
	return nil
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

func documentTypeNames(metadata *models.ExtractedMetadata, resultLanguage string) (displayName, originalName string) {
	originalName = strings.TrimSpace(metadata.DocumentType)
	if originalName == "" {
		return "", ""
	}
	if resultLanguage == "" {
		return originalName, originalName
	}

	translated := strings.TrimSpace(metadata.DocumentTypeTranslated)
	displayName = firstNonEmpty(translated, originalName)
	return displayName, originalName
}

func correspondentNames(metadata *models.ExtractedMetadata, resultLanguage string) (displayName, originalName string) {
	originalName = strings.TrimSpace(metadata.Correspondent)
	if originalName == "" {
		for _, raw := range metadata.PeopleOrOrganizations {
			if name := strings.TrimSpace(raw); name != "" {
				originalName = name
				break
			}
		}
	}
	if originalName == "" {
		return "", ""
	}
	if resultLanguage == "" {
		return originalName, originalName
	}

	translated := strings.TrimSpace(metadata.CorrespondentTranslated)
	displayName = firstNonEmpty(translated, originalName)
	return displayName, originalName
}

func applyCorrespondent(app core.App, document *core.Record, metadata *models.ExtractedMetadata, resultLanguage string) error {
	displayName, originalName := correspondentNames(metadata, resultLanguage)
	if displayName == "" {
		document.Set("correspondent", "")
		return nil
	}

	correspondentID, err := ensureCorrespondent(app, displayName, originalName)
	if err != nil {
		return err
	}
	document.Set("correspondent", correspondentID)
	return nil
}

func ensureCorrespondent(app core.App, displayName, originalName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	originalName = strings.TrimSpace(originalName)
	if displayName == "" {
		return "", nil
	}
	if originalName == "" {
		originalName = displayName
	}

	collection, err := app.FindCollectionByNameOrId("correspondents")
	if err != nil {
		return "", err
	}

	if id, err := findCorrespondentByOriginal(app, originalName); err != nil {
		return "", err
	} else if id != "" {
		return updateCorrespondentNames(app, id, displayName, originalName)
	}

	if id, err := findCorrespondentID(app, displayName); err != nil {
		return "", err
	} else if id != "" {
		return updateCorrespondentNames(app, id, displayName, originalName)
	}

	record := core.NewRecord(collection)
	record.Set("name", displayName)
	record.Set("name_original", originalName)
	if err := app.Save(record); err != nil {
		return "", err
	}
	return record.Id, nil
}

func updateCorrespondentNames(app core.App, id, displayName, originalName string) (string, error) {
	record, err := app.FindRecordById("correspondents", id)
	if err != nil {
		return "", err
	}

	changed := false
	if name := strings.TrimSpace(record.GetString("name")); name != displayName {
		record.Set("name", displayName)
		changed = true
	}
	if original := strings.TrimSpace(record.GetString("name_original")); original == "" || original != originalName {
		record.Set("name_original", originalName)
		changed = true
	}
	if changed {
		if err := app.Save(record); err != nil {
			return "", err
		}
	}
	return record.Id, nil
}

func findCorrespondentByOriginal(app core.App, originalName string) (string, error) {
	originalName = strings.TrimSpace(originalName)
	if originalName == "" {
		return "", nil
	}

	existing, err := app.FindRecordsByFilter(
		"correspondents",
		"name_original = {:name}",
		"",
		1,
		0,
		map[string]any{"name": originalName},
	)
	if err != nil {
		return "", err
	}
	if len(existing) == 0 {
		return "", nil
	}
	return existing[0].Id, nil
}

func findCorrespondentID(app core.App, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}

	existing, err := app.FindRecordsByFilter(
		"correspondents",
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

func applyDocumentType(app core.App, document *core.Record, metadata *models.ExtractedMetadata, resultLanguage string) error {
	displayName, originalName := documentTypeNames(metadata, resultLanguage)
	if displayName == "" {
		document.Set("document_type", "")
		return nil
	}

	typeID, err := ensureDocumentType(app, displayName, originalName)
	if err != nil {
		return err
	}
	document.Set("document_type", typeID)
	return nil
}

func ensureDocumentType(app core.App, displayName, originalName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	originalName = strings.TrimSpace(originalName)
	if displayName == "" {
		return "", nil
	}
	if originalName == "" {
		originalName = displayName
	}

	collection, err := app.FindCollectionByNameOrId("document_types")
	if err != nil {
		return "", err
	}

	if id, err := findDocumentTypeByOriginal(app, originalName); err != nil {
		return "", err
	} else if id != "" {
		return updateDocumentTypeNames(app, id, displayName, originalName)
	}

	if id, err := findDocumentTypeID(app, displayName); err != nil {
		return "", err
	} else if id != "" {
		return updateDocumentTypeNames(app, id, displayName, originalName)
	}

	record := core.NewRecord(collection)
	record.Set("name", displayName)
	record.Set("name_original", originalName)
	if err := app.Save(record); err != nil {
		return "", err
	}
	return record.Id, nil
}

func updateDocumentTypeNames(app core.App, id, displayName, originalName string) (string, error) {
	record, err := app.FindRecordById("document_types", id)
	if err != nil {
		return "", err
	}

	changed := false
	if name := strings.TrimSpace(record.GetString("name")); name != displayName {
		record.Set("name", displayName)
		changed = true
	}
	if original := strings.TrimSpace(record.GetString("name_original")); original == "" || original != originalName {
		record.Set("name_original", originalName)
		changed = true
	}
	if changed {
		if err := app.Save(record); err != nil {
			return "", err
		}
	}
	return record.Id, nil
}

func findDocumentTypeByOriginal(app core.App, originalName string) (string, error) {
	originalName = strings.TrimSpace(originalName)
	if originalName == "" {
		return "", nil
	}

	existing, err := app.FindRecordsByFilter(
		"document_types",
		"name_original = {:name}",
		"",
		1,
		0,
		map[string]any{"name": originalName},
	)
	if err != nil {
		return "", err
	}
	if len(existing) == 0 {
		return "", nil
	}
	return existing[0].Id, nil
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
	documentID := ""
	if document != nil {
		documentID = document.Id
	} else {
		documentID = job.GetString("document")
	}
	log.Printf("[worker] job=%s document=%s failed: %v", job.Id, documentID, err)

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

func truncateForLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
