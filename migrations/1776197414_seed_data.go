package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Seed icons (idempotent — skips existing)
		icons, err := app.FindCollectionByNameOrId("icons")
		if err != nil {
			return err
		}

		iconNames := []string{
			"school", "graduation-cap", "book-open", "books", "library",
			"building-2", "landmark", "file-text", "file-spreadsheet", "file-archive",
			"file-image", "file", "folder", "folder-open", "receipt",
			"file-check", "file-plus", "file-minus", "clipboard", "clipboard-check",
			"book", "notebook", "notebook-pen", "pencil", "pen",
			"printer", "scan", "search", "tag", "tags",
			"heart", "bookmark", "pin", "clock",
			"calendar", "calendar-days", "mail", "send", "inbox",
			"download", "upload", "cloud", "cloud-upload", "cloud-download",
			"image", "camera", "video", "music", "mic",
			"credit-card", "banknote", "coins", "wallet", "receipt-text",
			"shield", "shield-check", "lock", "unlock", "key",
			"user", "users", "user-check", "user-plus", "user-x",
			"settings", "sliders-horizontal", "list", "layout-grid", "table",
			"chart-bar", "chart-pie", "trending-up", "trending-down", "activity",
			"home", "house", "building", "store", "warehouse",
			"car", "truck", "plane", "train", "ship",
			"phone", "smartphone", "monitor", "laptop", "tablet",
			"globe", "map", "map-pin", "navigation", "compass",
			"zap", "sparkles", "crown", "award",
			"alert-circle", "alert-triangle", "info", "check-circle", "x-circle",
		}

		for _, name := range iconNames {
			existing, _ := app.FindFirstRecordByFilter(icons.Id, "name = {:name}", map[string]any{"name": name})
			if existing != nil {
				continue
			}
			record := core.NewRecord(icons)
			record.Set("name", name)
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: delete all icons
		icons, err := app.FindCollectionByNameOrId("icons")
		if err != nil {
			return nil
		}

		records, err := app.FindRecordsByFilter(icons.Id, "1=1", "", 0, 0)
		if err != nil {
			return nil
		}

		for _, record := range records {
			if err := app.Delete(record); err != nil {
				return err
			}
		}

		return nil
	})
}
