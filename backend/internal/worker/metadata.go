package worker

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/models"
)

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

	correspondentID, err := ensureNamedEntity(app, "correspondents", displayName, originalName)
	if err != nil {
		return err
	}
	document.Set("correspondent", correspondentID)
	return nil
}

func applyDocumentType(app core.App, document *core.Record, metadata *models.ExtractedMetadata, resultLanguage string) error {
	displayName, originalName := documentTypeNames(metadata, resultLanguage)
	if displayName == "" {
		document.Set("document_type", "")
		return nil
	}

	typeID, err := ensureNamedEntity(app, "document_types", displayName, originalName)
	if err != nil {
		return err
	}
	document.Set("document_type", typeID)
	return nil
}
