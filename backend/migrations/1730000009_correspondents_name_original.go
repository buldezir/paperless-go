package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		correspondents, err := app.FindCollectionByNameOrId("correspondents")
		if err != nil {
			return err
		}

		correspondents.Fields.Add(
			&core.TextField{Name: "name_original", Max: 255},
		)
		if err := app.Save(correspondents); err != nil {
			return err
		}

		records, err := app.FindRecordsByFilter("correspondents", "id != ''", "", 10000, 0, nil)
		if err != nil {
			return err
		}
		for _, record := range records {
			if strings.TrimSpace(record.GetString("name_original")) != "" {
				continue
			}
			name := strings.TrimSpace(record.GetString("name"))
			if name == "" {
				continue
			}
			record.Set("name_original", name)
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		correspondents, err := app.FindCollectionByNameOrId("correspondents")
		if err != nil {
			return err
		}

		correspondents.Fields.RemoveByName("name_original")
		return app.Save(correspondents)
	})
}
