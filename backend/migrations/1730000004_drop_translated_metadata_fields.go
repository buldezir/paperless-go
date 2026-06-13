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

		for _, name := range []string{
			"title_translated",
			"purpose_translated",
			"summary_translated",
		} {
			documents.Fields.RemoveByName(name)
		}

		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		documents.Fields.Add(
			&core.TextField{Name: "title_translated", Max: 500},
			&core.TextField{Name: "purpose_translated", Max: 1000},
			&core.TextField{Name: "summary_translated", Max: 5000},
		)

		return app.Save(documents)
	})
}
