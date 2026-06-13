package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("processing_jobs")
		if err != nil {
			return err
		}

		jobs.CreateRule = types.Pointer(`document.user = @request.auth.id && status = "pending"`)

		return app.Save(jobs)
	}, func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("processing_jobs")
		if err != nil {
			return err
		}

		jobs.CreateRule = nil

		return app.Save(jobs)
	})
}
