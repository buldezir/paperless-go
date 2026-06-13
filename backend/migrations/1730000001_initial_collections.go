package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		tags := core.NewBaseCollection("tags")
		tags.ListRule = types.Pointer("@request.auth.id != ''")
		tags.ViewRule = types.Pointer("@request.auth.id != ''")
		tags.CreateRule = types.Pointer("@request.auth.id != ''")
		tags.UpdateRule = types.Pointer("@request.auth.id != ''")
		tags.DeleteRule = types.Pointer("@request.auth.id != ''")
		tags.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 100},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		tags.AddIndex("idx_tags_name", true, "name", "")

		if err := app.Save(tags); err != nil {
			return err
		}

		documents := core.NewBaseCollection("documents")
		ownerRule := "user = @request.auth.id"
		documents.ListRule = types.Pointer(ownerRule)
		documents.ViewRule = types.Pointer(ownerRule)
		documents.CreateRule = types.Pointer(ownerRule)
		documents.UpdateRule = types.Pointer(ownerRule)
		documents.DeleteRule = types.Pointer(ownerRule)
		documents.Fields.Add(
			&core.FileField{
				Name:      "file",
				Required:  true,
				MaxSelect: 1,
				MaxSize:   20 << 20,
				MimeTypes: []string{
					"application/pdf",
					"image/jpeg",
					"image/png",
					"image/webp",
					"text/plain",
				},
			},
			&core.RelationField{
				Name:         "user",
				Required:     true,
				CollectionId: "_pb_users_auth_",
				MaxSelect:    1,
			},
			&core.TextField{Name: "title", Max: 500},
			&core.TextField{Name: "purpose", Max: 1000},
			&core.DateField{Name: "document_date"},
			&core.TextField{Name: "document_type", Max: 200},
			&core.EditorField{Name: "ocr_text"},
			&core.TextField{Name: "summary", Max: 5000},
			&core.SelectField{
				Name:   "processing_status",
				Values: []string{"pending", "processing", "completed", "failed", "needs_review"},
			},
			&core.TextField{Name: "metadata_source", Max: 200},
			&core.NumberField{Name: "confidence", Min: types.Pointer(0.0), Max: types.Pointer(1.0)},
			&core.JSONField{Name: "people_or_organizations"},
			&core.RelationField{
				Name:         "tags",
				CollectionId: tags.Id,
				MaxSelect:    50,
			},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)

		if err := app.Save(documents); err != nil {
			return err
		}

		jobs := core.NewBaseCollection("processing_jobs")
		jobs.ListRule = types.Pointer("document.user = @request.auth.id")
		jobs.ViewRule = types.Pointer("document.user = @request.auth.id")
		jobs.CreateRule = nil
		jobs.UpdateRule = nil
		jobs.DeleteRule = nil
		jobs.Fields.Add(
			&core.RelationField{
				Name:         "document",
				Required:     true,
				CollectionId: documents.Id,
				MaxSelect:    1,
			},
			&core.SelectField{
				Name:     "status",
				Required: true,
				Values:   []string{"pending", "running", "completed", "failed", "needs_review"},
			},
			&core.NumberField{Name: "retry_count", Min: types.Pointer(0.0)},
			&core.TextField{Name: "ocr_provider", Max: 100},
			&core.TextField{Name: "ai_provider", Max: 100},
			&core.TextField{Name: "prompt_version", Max: 50},
			&core.TextField{Name: "error_message", Max: 2000},
			&core.DateField{Name: "started_at"},
			&core.DateField{Name: "finished_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)

		return app.Save(jobs)
	}, func(app core.App) error {
		for _, name := range []string{"processing_jobs", "documents", "tags"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
		return nil
	})
}
