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

		documents.Fields.Add(
			&core.TextField{Name: "title_original", Max: 500},
			&core.TextField{Name: "purpose_original", Max: 1000},
			&core.TextField{Name: "summary_original", Max: 5000},
		)

		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		for _, name := range []string{
			"title_original",
			"purpose_original",
			"summary_original",
		} {
			documents.Fields.RemoveByName(name)
		}

		return app.Save(documents)
	})
}
