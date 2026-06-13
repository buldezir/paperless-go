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

		jobs.Fields.Add(
			&core.SelectField{
				Name:     "job_type",
				Required: false,
				Values:   []string{"full", "extraction"},
			},
		)

		records, err := app.FindRecordsByFilter("processing_jobs", "id != ''", "", 10000, 0, nil)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record.GetString("job_type") == "" {
				record.Set("job_type", "full")
				if err := app.Save(record); err != nil {
					return err
				}
			}
		}

		return app.Save(jobs)
	}, func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("processing_jobs")
		if err != nil {
			return err
		}

		jobs.Fields.RemoveByName("job_type")

		return app.Save(jobs)
	})
}
