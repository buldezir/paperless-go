package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		documents.Fields.RemoveByName("metadata_source")
		documents.Fields.Add(&core.TextField{Name: "metadata_source", Max: 200})

		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		documents.Fields.RemoveByName("metadata_source")
		documents.Fields.Add(&core.SelectField{
			Name:   "metadata_source",
			Values: []string{"ai", "user"},
		})

		return app.Save(documents)
	})
}
