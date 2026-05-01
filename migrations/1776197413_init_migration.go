package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		min0 := 0.0

		// ---------------------------------------------------------------------
		// Core (MVP) collections
		// ---------------------------------------------------------------------

		categories := core.NewBaseCollection("categories")
		categories.Fields.Add(
			&core.TextField{Name: "name", Required: true, Presentable: true},
			&core.TextField{Name: "description"},
			&core.JSONField{Name: "tags"},
			&core.TextField{Name: "color"},
			&core.TextField{Name: "icon"},
		)
		categories.Indexes = append(categories.Indexes,
			"CREATE UNIQUE INDEX `idx_categories_name` ON `categories` (`name`)",
		)

		if err := app.Save(categories); err != nil {
			return err
		}

		subcategories := core.NewBaseCollection("subcategories")
		subcategories.Fields.Add(
			&core.RelationField{Name: "category_id", CollectionId: categories.Id, Required: true, MaxSelect: 1},
			&core.TextField{Name: "name", Required: true, Presentable: true},
			&core.TextField{Name: "description"},
			&core.TextField{Name: "model_name", Required: true},
			&core.JSONField{Name: "tags"},
			&core.BoolField{Name: "is_default"},
		)
		subcategories.Indexes = append(subcategories.Indexes,
			"CREATE UNIQUE INDEX `idx_subcategories_category_name` ON `subcategories` (`category_id`, `name`)",
			"CREATE UNIQUE INDEX `idx_subcategories_default_per_category` ON `subcategories` (`category_id`) WHERE `is_default` = TRUE",
		)

		if err := app.Save(subcategories); err != nil {
			return err
		}

		documents := core.NewBaseCollection("documents")
		documents.Fields.Add(
			&core.TextField{Name: "name", Required: true, Presentable: true},
			&core.TextField{Name: "file_name", Required: true},
			&core.TextField{Name: "file_ext", Required: true},
			&core.NumberField{Name: "file_size", Min: &min0, OnlyInt: true},

			// context_menu -> path (local)
			&core.TextField{Name: "path"},

			// manual_upload (app) -> file stored in PocketBase
			&core.FileField{Name: "file", MaxSelect: 1},

			&core.TextField{Name: "hash"},
			&core.EditorField{Name: "ocr_txt"},
			&core.JSONField{Name: "metadata"},
			&core.RelationField{Name: "category_id", CollectionId: categories.Id, Required: true, MaxSelect: 1},
			&core.RelationField{Name: "subcategory_id", CollectionId: subcategories.Id, Required: true, MaxSelect: 1},
			&core.SelectField{Name: "status", Required: true, Values: []string{"pending", "processed", "error"}},
			&core.SelectField{Name: "source_type", Values: []string{"context_menu", "manual_upload", "drag_drop"}},
			&core.EditorField{Name: "notes"},
		)
		documents.Indexes = append(documents.Indexes,
			"CREATE UNIQUE INDEX `idx_documents_hash` ON `documents` (`hash`) WHERE `hash` != ''",
		)

		if err := app.Save(documents); err != nil {
			return err
		}

		documentTags := core.NewBaseCollection("document_tags")
		documentTags.Fields.Add(
			&core.RelationField{Name: "document_id", CollectionId: documents.Id, Required: true, MaxSelect: 1},
			&core.TextField{Name: "tag", Required: true},
		)
		documentTags.Indexes = append(documentTags.Indexes,
			"CREATE UNIQUE INDEX `idx_document_tags_document_tag` ON `document_tags` (`document_id`, `tag`)",
		)

		if err := app.Save(documentTags); err != nil {
			return err
		}

		imports := core.NewBaseCollection("imports")
		imports.Fields.Add(
			&core.TextField{Name: "source_path"},
			&core.SelectField{Name: "source_type", Required: true, Values: []string{"context_menu", "manual_upload", "drag_drop"}},
			&core.NumberField{Name: "total_files", Min: &min0, OnlyInt: true},
			&core.NumberField{Name: "processed_files", Min: &min0, OnlyInt: true},
			&core.SelectField{Name: "status", Required: true, Values: []string{"pending", "processing", "done", "error"}},
			&core.EditorField{Name: "error_message"},
		)

		if err := app.Save(imports); err != nil {
			return err
		}

		importItems := core.NewBaseCollection("import_items")
		importItems.Fields.Add(
			&core.RelationField{Name: "import_id", CollectionId: imports.Id, Required: true, MaxSelect: 1},
			&core.RelationField{Name: "document_id", CollectionId: documents.Id, MaxSelect: 1},
			&core.TextField{Name: "original_path", Required: true},
			&core.SelectField{Name: "status", Required: true, Values: []string{"pending", "processed", "skipped", "error"}},
			&core.EditorField{Name: "error_message"},
		)

		if err := app.Save(importItems); err != nil {
			return err
		}

		appSettings := core.NewBaseCollection("app_settings")
		appSettings.Fields.Add(
			&core.TextField{Name: "key", Required: true},
			&core.JSONField{Name: "value", Required: true},
			&core.TextField{Name: "description"},
		)
		appSettings.Indexes = append(appSettings.Indexes,
			"CREATE UNIQUE INDEX `idx_app_settings_key` ON `app_settings` (`key`)",
		)

		if err := app.Save(appSettings); err != nil {
			return err
		}

		// ---------------------------------------------------------------------
		// Optional extensions
		// ---------------------------------------------------------------------

		folders := core.NewBaseCollection("folders")
		folders.Fields.Add(
			&core.TextField{Name: "name", Required: true, Presentable: true},
			&core.TextField{Name: "description"},
		)

		// Save first, then add self-relation (RelationField requires an existing collectionId).
		if err := app.Save(folders); err != nil {
			return err
		}

		folders.Fields.Add(
			&core.RelationField{Name: "parent_id", CollectionId: folders.Id, MaxSelect: 1},
		)

		if err := app.Save(folders); err != nil {
			return err
		}

		documentFolders := core.NewBaseCollection("document_folders")
		documentFolders.Fields.Add(
			&core.RelationField{Name: "document_id", CollectionId: documents.Id, Required: true, MaxSelect: 1},
			&core.RelationField{Name: "folder_id", CollectionId: folders.Id, Required: true, MaxSelect: 1},
		)
		documentFolders.Indexes = append(documentFolders.Indexes,
			"CREATE UNIQUE INDEX `idx_document_folders_document_folder` ON `document_folders` (`document_id`, `folder_id`)",
		)

		if err := app.Save(documentFolders); err != nil {
			return err
		}

		savedFilters := core.NewBaseCollection("saved_filters")
		savedFilters.Fields.Add(
			&core.TextField{Name: "name", Required: true, Presentable: true},
			&core.JSONField{Name: "filter_config", Required: true},
		)

		if err := app.Save(savedFilters); err != nil {
			return err
		}

		documentLinks := core.NewBaseCollection("document_links")
		documentLinks.Fields.Add(
			&core.RelationField{Name: "document_id", CollectionId: documents.Id, Required: true, MaxSelect: 1},
			&core.RelationField{Name: "linked_document_id", CollectionId: documents.Id, Required: true, MaxSelect: 1},
			&core.SelectField{Name: "relation_type", Required: true, Values: []string{"related", "attachment", "reference"}},
		)
		documentLinks.Indexes = append(documentLinks.Indexes,
			"CREATE UNIQUE INDEX `idx_document_links_unique` ON `document_links` (`document_id`, `linked_document_id`, `relation_type`)",
		)

		if err := app.Save(documentLinks); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// delete in reverse order to avoid relation dependency issues
		for _, name := range []string{
			"document_links",
			"saved_filters",
			"document_folders",
			"folders",
			"app_settings",
			"import_items",
			"imports",
			"document_tags",
			"documents",
			"subcategories",
			"categories",
		} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}

			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
