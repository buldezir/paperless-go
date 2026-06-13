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

		_, err := createProcessingJob(e.App, record.Id)
		return err
	})

	app.OnRecordCreate("processing_jobs").BindFunc(func(e *core.RecordEvent) error {
		record := e.Record
		if record.GetString("task_id") == "" {
			record.Set("task_id", uuid.New().String())
		}
		if record.Get("retry_count") == nil {
			record.Set("retry_count", 0)
		}
		return e.Next()
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

func createProcessingJob(app core.App, documentID string) (*core.Record, error) {
	jobsCollection, err := app.FindCollectionByNameOrId("processing_jobs")
	if err != nil {
		return nil, err
	}

	job := core.NewRecord(jobsCollection)
	job.Set("document", documentID)
	job.Set("status", models.JobStatusPending)

	if err := app.Save(job); err != nil {
		return nil, err
	}

	return job, nil
}
