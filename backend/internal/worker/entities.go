package worker

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func ensureNamedEntity(app core.App, collection, displayName, originalName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	originalName = strings.TrimSpace(originalName)
	if displayName == "" {
		return "", nil
	}
	if originalName == "" {
		originalName = displayName
	}

	if id, err := findNamedEntity(app, collection, "name_original", originalName); err != nil {
		return "", err
	} else if id != "" {
		return updateNamedEntity(app, collection, id, displayName, originalName)
	}

	if id, err := findNamedEntity(app, collection, "name", displayName); err != nil {
		return "", err
	} else if id != "" {
		return updateNamedEntity(app, collection, id, displayName, originalName)
	}

	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		return "", err
	}

	record := core.NewRecord(coll)
	record.Set("name", displayName)
	record.Set("name_original", originalName)
	if err := app.Save(record); err != nil {
		return "", err
	}
	return record.Id, nil
}

func findNamedEntity(app core.App, collection, field, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	existing, err := app.FindRecordsByFilter(
		collection,
		field+" = {:name}",
		"",
		1,
		0,
		map[string]any{"name": value},
	)
	if err != nil {
		return "", err
	}
	if len(existing) == 0 {
		return "", nil
	}
	return existing[0].Id, nil
}

func updateNamedEntity(app core.App, collection, id, displayName, originalName string) (string, error) {
	record, err := app.FindRecordById(collection, id)
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
