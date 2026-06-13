package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		documentTypes, err := app.FindCollectionByNameOrId("document_types")
		if err != nil {
			return err
		}

		records, err := app.FindRecordsByFilter("documents", "id != ''", "", 10000, 0, nil)
		if err != nil {
			return err
		}

		type docMigration struct {
			id           string
			ngxTypeID    string
			textTypeName string
		}
		pending := make([]docMigration, 0, len(records))
		for _, record := range records {
			pending = append(pending, docMigration{
				id:           record.Id,
				ngxTypeID:    record.GetString("ngx_document_type"),
				textTypeName: strings.TrimSpace(record.GetString("document_type")),
			})
		}

		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		documents.Fields.RemoveByName("document_type")
		documents.Fields.RemoveByName("ngx_document_type")
		documents.Fields.Add(
			&core.RelationField{
				Name:         "document_type",
				CollectionId: documentTypes.Id,
				MaxSelect:    1,
			},
		)
		if err := app.Save(documents); err != nil {
			return err
		}

		for _, item := range pending {
			record, err := app.FindRecordById("documents", item.id)
			if err != nil {
				return err
			}

			var typeID string
			switch {
			case item.ngxTypeID != "":
				typeID = item.ngxTypeID
			case item.textTypeName != "":
				typeID, err = findOrCreateDocumentType(app, documentTypes, item.textTypeName)
				if err != nil {
					return err
				}
			}

			if typeID != "" {
				record.Set("document_type", typeID)
				if err := app.Save(record); err != nil {
					return err
				}
			}
		}

		return nil
	}, func(app core.App) error {
		documentTypes, err := app.FindCollectionByNameOrId("document_types")
		if err != nil {
			return err
		}

		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		documents.Fields.RemoveByName("document_type")
		documents.Fields.Add(
			&core.TextField{Name: "document_type", Max: 200},
			&core.RelationField{
				Name:         "ngx_document_type",
				CollectionId: documentTypes.Id,
				MaxSelect:    1,
			},
		)
		return app.Save(documents)
	})
}

func findOrCreateDocumentType(app core.App, collection *core.Collection, name string) (string, error) {
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
	if len(existing) > 0 {
		return existing[0].Id, nil
	}

	record := core.NewRecord(collection)
	record.Set("name", name)
	if err := app.Save(record); err != nil {
		return "", err
	}
	return record.Id, nil
}
