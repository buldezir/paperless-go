package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		correspondents := core.NewBaseCollection("correspondents")
		authRule := "@request.auth.id != ''"
		correspondents.ListRule = types.Pointer(authRule)
		correspondents.ViewRule = types.Pointer(authRule)
		correspondents.CreateRule = types.Pointer(authRule)
		correspondents.UpdateRule = types.Pointer(authRule)
		correspondents.DeleteRule = types.Pointer(authRule)
		correspondents.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		correspondents.AddIndex("idx_correspondents_name", true, "name", "")
		if err := app.Save(correspondents); err != nil {
			return err
		}

		documentTypes := core.NewBaseCollection("document_types")
		documentTypes.ListRule = types.Pointer(authRule)
		documentTypes.ViewRule = types.Pointer(authRule)
		documentTypes.CreateRule = types.Pointer(authRule)
		documentTypes.UpdateRule = types.Pointer(authRule)
		documentTypes.DeleteRule = types.Pointer(authRule)
		documentTypes.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		documentTypes.AddIndex("idx_document_types_name", true, "name", "")
		if err := app.Save(documentTypes); err != nil {
			return err
		}

		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		documents.Fields.Add(
			&core.RelationField{
				Name:         "correspondent",
				CollectionId: correspondents.Id,
				MaxSelect:    1,
			},
			&core.RelationField{
				Name:         "ngx_document_type",
				CollectionId: documentTypes.Id,
				MaxSelect:    1,
			},
		)
		if err := app.Save(documents); err != nil {
			return err
		}

		jobs, err := app.FindCollectionByNameOrId("processing_jobs")
		if err != nil {
			return err
		}
		jobs.Fields.Add(
			&core.TextField{Name: "task_id", Max: 36},
		)
		jobs.AddIndex("idx_processing_jobs_task_id", false, "task_id", "")
		return app.Save(jobs)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err == nil {
			documents.Fields.RemoveByName("correspondent")
			documents.Fields.RemoveByName("ngx_document_type")
			if err := app.Save(documents); err != nil {
				return err
			}
		}

		jobs, err := app.FindCollectionByNameOrId("processing_jobs")
		if err == nil {
			jobs.Fields.RemoveByName("task_id")
			jobs.RemoveIndex("idx_processing_jobs_task_id")
			if err := app.Save(jobs); err != nil {
				return err
			}
		}

		for _, name := range []string{"document_types", "correspondents"} {
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
