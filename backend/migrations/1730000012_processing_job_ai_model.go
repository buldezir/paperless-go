package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("processing_jobs")
		if err != nil {
			return err
		}

		jobs.Fields.Add(&core.TextField{Name: "ai_model", Max: 200})

		return app.Save(jobs)
	}, func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("processing_jobs")
		if err != nil {
			return err
		}

		jobs.Fields.RemoveByName("ai_model")

		return app.Save(jobs)
	})
}
