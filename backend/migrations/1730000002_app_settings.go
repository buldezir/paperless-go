package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		settings := core.NewBaseCollection("app_settings")
		// Locked down for regular users; superusers bypass rules.
		// The React app uses /api/app/settings instead of the collection API.
		settings.Fields.Add(
			&core.TextField{Name: "ocr_provider", Max: 50},
			&core.TextField{Name: "google_vision_api_key", Max: 2000},
			&core.TextField{Name: "mistral_api_key", Max: 2000},
			&core.TextField{Name: "mistral_ocr_model", Max: 200},
			&core.TextField{Name: "mistral_api_base_url", Max: 500},
			&core.NumberField{Name: "ocr_timeout_sec", OnlyInt: true},
			&core.TextField{Name: "processing_result_language", Max: 16},
			&core.TextField{Name: "openai_api_key", Max: 2000},
			&core.TextField{Name: "openai_model", Max: 200},
			&core.TextField{Name: "openai_chat_model", Max: 200},
			&core.TextField{Name: "openai_base_url", Max: 500},
			&core.NumberField{Name: "openai_timeout_sec", OnlyInt: true},
			&core.NumberField{Name: "worker_timeout_sec", OnlyInt: true},
			&core.NumberField{Name: "worker_max_retries", OnlyInt: true},
			&core.TextField{Name: "extraction_prompt_version", Max: 50},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		return app.Save(settings)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			return nil
		}
		return app.Delete(collection)
	})
}
