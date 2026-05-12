package migrations

import (
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/core"
)

func init() {
	m.Register(func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		// Check if thumbnail field already exists
		for _, f := range documents.Fields {
			if f.GetName() == "thumbnail" {
				return nil
			}
		}

		documents.Fields.Add(&core.FileField{Name: "thumbnail", MaxSelect: 1})
		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		documents.Fields.RemoveByName("thumbnail")
		return app.Save(documents)
	})
}
