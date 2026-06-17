package worker

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/models"
	"paperless-go/backend/internal/ocr"
)

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

	return tmpPath, ocr.GuessMimeType(fileName), cleanup, nil
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

func stepsInclude(steps []string, name string) bool {
	for _, step := range steps {
		if step == name {
			return true
		}
	}
	return false
}

func finalizeDocumentWithoutApply(app core.App, document *core.Record, steps []string) error {
	if stepsInclude(steps, models.StepApplyMetadata) {
		return nil
	}
	document.Set("processing_status", models.DocStatusCompleted)
	return app.Save(document)
}
