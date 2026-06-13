package hooks

import (
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"paperless-go/backend/internal/models"
)

func Register(app core.App) {
	app.OnRecordCreate("documents").BindFunc(func(e *core.RecordEvent) error {
		record := e.Record
		if record.GetString("processing_status") == "" {
			record.Set("processing_status", models.DocStatusPending)
		}
		if record.GetString("metadata_source") == "" {
			record.Set("metadata_source", models.MetadataSourceAI)
		}

		if err := e.Next(); err != nil {
			return err
		}

		jobsCollection, err := e.App.FindCollectionByNameOrId("processing_jobs")
		if err != nil {
			return err
		}

		job := core.NewRecord(jobsCollection)
		job.Set("document", record.Id)
		job.Set("status", models.JobStatusPending)
		job.Set("retry_count", 0)
		job.Set("task_id", uuid.New().String())

		return e.App.Save(job)
	})

	app.OnRecordDelete("documents").BindFunc(func(e *core.RecordEvent) error {
		jobs, err := e.App.FindRecordsByFilter(
			"processing_jobs",
			"document = {:docId}",
			"-created",
			100,
			0,
			map[string]any{"docId": e.Record.Id},
		)
		if err != nil {
			return err
		}

		for _, job := range jobs {
			if err := e.App.Delete(job); err != nil {
				return err
			}
		}

		return e.Next()
	})
}
