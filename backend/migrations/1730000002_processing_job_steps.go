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
			&core.JSONField{Name: "steps"},
			&core.JSONField{Name: "step_runs"},
			&core.TextField{Name: "current_step", Max: 50},
			&core.JSONField{Name: "metadata_json"},
			&core.JSONField{Name: "force_steps"},
		)
		if err := app.Save(jobs); err != nil {
			return err
		}

		records, err := app.FindAllRecords("processing_jobs")
		if err != nil {
			return err
		}

		for _, job := range records {
			if job.Get("steps") != nil {
				continue
			}

			job.Set("steps", []string{"preview", "ocr", "extract_metadata", "apply_metadata"})
			if err := app.Save(job); err != nil {
				return err
			}
		}

		for _, name := range []string{
			"job_type",
			"retry_count",
			"ocr_provider",
			"ai_provider",
			"ai_model",
			"prompt_version",
			"error_message",
		} {
			jobs.Fields.RemoveByName(name)
		}
		return app.Save(jobs)
	}, func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("processing_jobs")
		if err != nil {
			return err
		}

		jobs.Fields.Add(
			&core.SelectField{
				Name:   "job_type",
				Values: []string{"full", "extraction"},
			},
			&core.NumberField{Name: "retry_count"},
			&core.TextField{Name: "ocr_provider", Max: 100},
			&core.TextField{Name: "ai_provider", Max: 100},
			&core.TextField{Name: "ai_model", Max: 200},
			&core.TextField{Name: "prompt_version", Max: 50},
			&core.TextField{Name: "error_message", Max: 2000},
		)
		for _, name := range []string{"steps", "step_runs", "current_step", "metadata_json", "force_steps"} {
			jobs.Fields.RemoveByName(name)
		}
		return app.Save(jobs)
	})
}
