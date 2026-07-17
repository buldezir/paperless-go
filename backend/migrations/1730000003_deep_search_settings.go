package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("deep_search_languages") == nil {
			collection.Fields.Add(&core.TextField{Name: "deep_search_languages", Max: 200})
		}
		if collection.Fields.GetByName("openai_search_model") == nil {
			collection.Fields.Add(&core.TextField{Name: "openai_search_model", Max: 200})
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			return nil
		}
		if f := collection.Fields.GetByName("deep_search_languages"); f != nil {
			collection.Fields.RemoveById(f.GetId())
		}
		if f := collection.Fields.GetByName("openai_search_model"); f != nil {
			collection.Fields.RemoveById(f.GetId())
		}
		return app.Save(collection)
	})
}
