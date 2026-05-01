package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	m "github.com/pocketbase/pocketbase/migrations"
)

func ensureCollection(app core.App, name string) (*core.Collection, bool, error) {
	col, err := app.FindCollectionByNameOrId(name)
	if err == nil {
		return col, false, nil
	}
	return core.NewBaseCollection(name), true, nil
}

func setPublicRules(c *core.Collection) {
	r := types.Pointer("")
	c.ListRule = r
	c.ViewRule = r
	c.CreateRule = r
	c.UpdateRule = r
	c.DeleteRule = r
}

func setReadOnlyRules(c *core.Collection) {
	r := types.Pointer("")
	c.ListRule = r
	c.ViewRule = r
	c.CreateRule = nil
	c.UpdateRule = nil
	c.DeleteRule = nil
}

func addField(c *core.Collection, field core.Field) {
	for _, f := range c.Fields {
		if f.GetName() == field.GetName() {
			return
		}
	}
	c.Fields.Add(field)
}

func addAutodateFields(c *core.Collection) {
	for _, f := range c.Fields {
		if f.GetName() == "created" || f.GetName() == "updated" {
			return
		}
	}
	c.Fields.AddMarshaledJSONAt(len(c.Fields), []byte(`{
		"hidden": false,
		"id": "",
		"name": "created",
		"onCreate": true,
		"onUpdate": false,
		"presentable": false,
		"system": false,
		"type": "autodate"
	}`))
	c.Fields.AddMarshaledJSONAt(len(c.Fields), []byte(`{
		"hidden": false,
		"id": "",
		"name": "updated",
		"onCreate": true,
		"onUpdate": true,
		"presentable": false,
		"system": false,
		"type": "autodate"
	}`))
}

func init() {
	m.Register(func(app core.App) error {
		min0 := 0.0

		// =====================================================================
		// PHASE 1: Create all collections with simple fields only (no relations)
		// =====================================================================

		icons, isNew, err := ensureCollection(app, "icons")
		if err != nil {
			return err
		}
		if isNew {
			icons.Fields.Add(
				&core.TextField{Name: "name", Required: true, Presentable: true},
				&core.TextField{Name: "label"},
			)
		}
		setPublicRules(icons)
		addAutodateFields(icons)
		if err := app.Save(icons); err != nil {
			return err
		}

		categories, isNew, err := ensureCollection(app, "categories")
		if err != nil {
			return err
		}
		if isNew {
			categories.Fields.Add(
				&core.TextField{Name: "name", Required: true, Presentable: true},
				&core.TextField{Name: "description"},
				&core.JSONField{Name: "tags"},
				&core.TextField{Name: "color"},
			)
		}
		setPublicRules(categories)
		addAutodateFields(categories)
		if err := app.Save(categories); err != nil {
			return err
		}

		subcategories, isNew, err := ensureCollection(app, "subcategories")
		if err != nil {
			return err
		}
		if isNew {
			subcategories.Fields.Add(
				&core.TextField{Name: "name", Required: true, Presentable: true},
				&core.TextField{Name: "description"},
				&core.TextField{Name: "model_name", Required: true},
				&core.JSONField{Name: "tags"},
				&core.BoolField{Name: "is_default"},
			)
		}
		setPublicRules(subcategories)
		addAutodateFields(subcategories)
		if err := app.Save(subcategories); err != nil {
			return err
		}

		documents, isNew, err := ensureCollection(app, "documents")
		if err != nil {
			return err
		}
		if isNew {
			documents.Fields.Add(
				&core.TextField{Name: "name", Required: true, Presentable: true},
				&core.TextField{Name: "file_name", Required: true},
				&core.TextField{Name: "file_ext", Required: true},
				&core.NumberField{Name: "file_size", Min: &min0, OnlyInt: true},
				&core.TextField{Name: "path"},
				&core.FileField{Name: "file", MaxSelect: 1},
				&core.TextField{Name: "hash"},
				&core.EditorField{Name: "ocr_txt"},
				&core.JSONField{Name: "metadata"},
				&core.SelectField{Name: "status", Required: true, Values: []string{"pending", "processed", "error"}},
				&core.SelectField{Name: "source_type", Values: []string{"context_menu", "manual_upload", "drag_drop"}},
				&core.EditorField{Name: "notes"},
				&core.DateField{Name: "last_access"},
				&core.BoolField{Name: "is_favorite"},
			)
		}
		setReadOnlyRules(documents)
		addAutodateFields(documents)
		if err := app.Save(documents); err != nil {
			return err
		}

		documentTags, isNew, err := ensureCollection(app, "document_tags")
		if err != nil {
			return err
		}
		if isNew {
			documentTags.Fields.Add(
				&core.TextField{Name: "tag", Required: true},
			)
		}
		setPublicRules(documentTags)
		addAutodateFields(documentTags)
		if err := app.Save(documentTags); err != nil {
			return err
		}

		imports, isNew, err := ensureCollection(app, "imports")
		if err != nil {
			return err
		}
		if isNew {
			imports.Fields.Add(
				&core.TextField{Name: "source_path"},
				&core.SelectField{Name: "source_type", Required: true, Values: []string{"context_menu", "manual_upload", "drag_drop"}},
				&core.NumberField{Name: "total_files", Min: &min0, OnlyInt: true},
				&core.NumberField{Name: "processed_files", Min: &min0, OnlyInt: true},
				&core.SelectField{Name: "status", Required: true, Values: []string{"pending", "processing", "done", "error"}},
				&core.EditorField{Name: "error_message"},
			)
		}
		setPublicRules(imports)
		addAutodateFields(imports)
		if err := app.Save(imports); err != nil {
			return err
		}

		importItems, isNew, err := ensureCollection(app, "import_items")
		if err != nil {
			return err
		}
		if isNew {
			importItems.Fields.Add(
				&core.TextField{Name: "original_path", Required: true},
				&core.SelectField{Name: "status", Required: true, Values: []string{"pending", "processed", "skipped", "error"}},
				&core.EditorField{Name: "error_message"},
			)
		}
		setPublicRules(importItems)
		addAutodateFields(importItems)
		if err := app.Save(importItems); err != nil {
			return err
		}

		appSettings, isNew, err := ensureCollection(app, "app_settings")
		if err != nil {
			return err
		}
		if isNew {
			appSettings.Fields.Add(
				&core.TextField{Name: "key", Required: true},
				&core.JSONField{Name: "value", Required: true},
				&core.TextField{Name: "description"},
			)
		}
		setPublicRules(appSettings)
		addAutodateFields(appSettings)
		if err := app.Save(appSettings); err != nil {
			return err
		}

		folders, isNew, err := ensureCollection(app, "folders")
		if err != nil {
			return err
		}
		if isNew {
			folders.Fields.Add(
				&core.TextField{Name: "name", Required: true, Presentable: true},
				&core.TextField{Name: "description"},
			)
		}
		setPublicRules(folders)
		addAutodateFields(folders)
		if err := app.Save(folders); err != nil {
			return err
		}

		documentFolders, isNew, err := ensureCollection(app, "document_folders")
		if err != nil {
			return err
		}
		setPublicRules(documentFolders)
		addAutodateFields(documentFolders)
		if err := app.Save(documentFolders); err != nil {
			return err
		}

		savedFilters, isNew, err := ensureCollection(app, "saved_filters")
		if err != nil {
			return err
		}
		if isNew {
			savedFilters.Fields.Add(
				&core.TextField{Name: "name", Required: true, Presentable: true},
				&core.JSONField{Name: "filter_config", Required: true},
			)
		}
		setPublicRules(savedFilters)
		addAutodateFields(savedFilters)
		if err := app.Save(savedFilters); err != nil {
			return err
		}

		documentLinks, isNew, err := ensureCollection(app, "document_links")
		if err != nil {
			return err
		}
		if isNew {
			documentLinks.Fields.Add(
				&core.TextField{Name: "relation_type", Required: true},
			)
		} else {
			addField(documentLinks, &core.TextField{Name: "relation_type", Required: true})
		}
		setPublicRules(documentLinks)
		addAutodateFields(documentLinks)
		if err := app.Save(documentLinks); err != nil {
			return err
		}

		// =====================================================================
		// PHASE 2: Refresh all collections to get correct IDs
		// =====================================================================

		icons, err = app.FindCollectionByNameOrId("icons")
		if err != nil {
			return err
		}
		categories, err = app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}
		subcategories, err = app.FindCollectionByNameOrId("subcategories")
		if err != nil {
			return err
		}
		documents, err = app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		documentTags, err = app.FindCollectionByNameOrId("document_tags")
		if err != nil {
			return err
		}
		imports, err = app.FindCollectionByNameOrId("imports")
		if err != nil {
			return err
		}
		importItems, err = app.FindCollectionByNameOrId("import_items")
		if err != nil {
			return err
		}
		appSettings, err = app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			return err
		}
		folders, err = app.FindCollectionByNameOrId("folders")
		if err != nil {
			return err
		}
		documentFolders, err = app.FindCollectionByNameOrId("document_folders")
		if err != nil {
			return err
		}
		savedFilters, err = app.FindCollectionByNameOrId("saved_filters")
		if err != nil {
			return err
		}
		documentLinks, err = app.FindCollectionByNameOrId("document_links")
		if err != nil {
			return err
		}

		// =====================================================================
		// PHASE 3: Add relation fields (now all collection IDs are valid)
		// =====================================================================

		addField(categories, &core.RelationField{Name: "icon_id", CollectionId: icons.Id, MaxSelect: 1})
		if err := app.Save(categories); err != nil {
			return err
		}

		addField(subcategories, &core.RelationField{Name: "category_id", CollectionId: categories.Id, Required: true, MaxSelect: 1})
		if err := app.Save(subcategories); err != nil {
			return err
		}

		addField(documents, &core.RelationField{Name: "category_id", CollectionId: categories.Id, Required: true, MaxSelect: 1})
		addField(documents, &core.RelationField{Name: "subcategory_id", CollectionId: subcategories.Id, Required: true, MaxSelect: 1})
		addField(documents, &core.DateField{Name: "last_access"})
		addField(documents, &core.BoolField{Name: "is_favorite"})
		if err := app.Save(documents); err != nil {
			return err
		}

		addField(documentTags, &core.RelationField{Name: "document_id", CollectionId: documents.Id, Required: true, MaxSelect: 1})
		if err := app.Save(documentTags); err != nil {
			return err
		}

		addField(importItems, &core.RelationField{Name: "import_id", CollectionId: imports.Id, Required: true, MaxSelect: 1})
		addField(importItems, &core.RelationField{Name: "document_id", CollectionId: documents.Id, MaxSelect: 1})
		if err := app.Save(importItems); err != nil {
			return err
		}

		addField(folders, &core.RelationField{Name: "parent_id", CollectionId: folders.Id, MaxSelect: 1})
		if err := app.Save(folders); err != nil {
			return err
		}

		addField(documentFolders, &core.RelationField{Name: "document_id", CollectionId: documents.Id, Required: true, MaxSelect: 1})
		addField(documentFolders, &core.RelationField{Name: "folder_id", CollectionId: folders.Id, Required: true, MaxSelect: 1})
		if err := app.Save(documentFolders); err != nil {
			return err
		}

		addField(documentLinks, &core.RelationField{Name: "document_id", CollectionId: documents.Id, Required: true, MaxSelect: 1})
		addField(documentLinks, &core.RelationField{Name: "linked_document_id", CollectionId: documents.Id, Required: true, MaxSelect: 1})
		if err := app.Save(documentLinks); err != nil {
			return err
		}

		// =====================================================================
		// PHASE 4: Add indexes (after all fields including relations exist)
		// =====================================================================

		icons, err = app.FindCollectionByNameOrId("icons")
		if err != nil {
			return err
		}
		icons.Indexes = append(icons.Indexes,
			"CREATE UNIQUE INDEX `idx_icons_name` ON `icons` (`name`)",
		)
		if err := app.Save(icons); err != nil {
			return err
		}

		categories, err = app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}
		categories.Indexes = append(categories.Indexes,
			"CREATE UNIQUE INDEX `idx_categories_name` ON `categories` (`name`)",
		)
		if err := app.Save(categories); err != nil {
			return err
		}

		subcategories, err = app.FindCollectionByNameOrId("subcategories")
		if err != nil {
			return err
		}
		subcategories.Indexes = append(subcategories.Indexes,
			"CREATE UNIQUE INDEX `idx_subcategories_category_name` ON `subcategories` (`category_id`, `name`)",
			"CREATE UNIQUE INDEX `idx_subcategories_default_per_category` ON `subcategories` (`category_id`) WHERE `is_default` = TRUE",
		)
		if err := app.Save(subcategories); err != nil {
			return err
		}

		documents, err = app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		documents.Indexes = append(documents.Indexes,
			"CREATE UNIQUE INDEX `idx_documents_hash` ON `documents` (`hash`) WHERE `hash` != ''",
		)
		if err := app.Save(documents); err != nil {
			return err
		}

		documentTags, err = app.FindCollectionByNameOrId("document_tags")
		if err != nil {
			return err
		}
		documentTags.Indexes = append(documentTags.Indexes,
			"CREATE UNIQUE INDEX `idx_document_tags_document_tag` ON `document_tags` (`document_id`, `tag`)",
		)
		if err := app.Save(documentTags); err != nil {
			return err
		}

		appSettings, err = app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			return err
		}
		appSettings.Indexes = append(appSettings.Indexes,
			"CREATE UNIQUE INDEX `idx_app_settings_key` ON `app_settings` (`key`)",
		)
		if err := app.Save(appSettings); err != nil {
			return err
		}

		documentFolders, err = app.FindCollectionByNameOrId("document_folders")
		if err != nil {
			return err
		}
		documentFolders.Indexes = append(documentFolders.Indexes,
			"CREATE UNIQUE INDEX `idx_document_folders_document_folder` ON `document_folders` (`document_id`, `folder_id`)",
		)
		if err := app.Save(documentFolders); err != nil {
			return err
		}

		documentLinks, err = app.FindCollectionByNameOrId("document_links")
		if err != nil {
			return err
		}
		documentLinks.Indexes = append(documentLinks.Indexes,
			"CREATE UNIQUE INDEX `idx_document_links_unique` ON `document_links` (`document_id`, `linked_document_id`, `relation_type`)",
		)
		if err := app.Save(documentLinks); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Delete in reverse dependency order
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
			"icons",
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