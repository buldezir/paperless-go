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
			&core.FileField{
				Name:      "preview",
				Required:  false,
				MaxSelect: 1,
				MaxSize:   2 << 20,
				MimeTypes: []string{
					"image/png",
				},
			},
		)

		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		documents.Fields.RemoveByName("preview")

		return app.Save(documents)
	})
}
