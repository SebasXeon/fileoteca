package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"syscall"
	"unsafe"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"SebasXeon/Fileoteca/internal/addfile"
)

func msgBox(msg string) {
	caption, _ := syscall.UTF16PtrFromString("Fileoteca")
	text, _ := syscall.UTF16PtrFromString(msg)
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(
		0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(caption)), 0x10,
	)
}

func ShowError(msg string) {
	msgBox(msg)
}

func ShowInfo(msg string) {
	caption, _ := syscall.UTF16PtrFromString("Fileoteca")
	text, _ := syscall.UTF16PtrFromString(msg)
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(
		0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(caption)), 0x40,
	)
}

func AddFileViaHTTP(info *addfile.Info) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("no se pudo leer config: %w", err)
	}
	if cfg.DefaultCategoryID == "" || cfg.DefaultSubcategoryID == "" {
		return fmt.Errorf("IDs de categoría por defecto no configurados — inicia la app normalmente primero")
	}

	body := map[string]any{
		"name":           info.Name,
		"file_name":      info.FileName,
		"file_ext":       info.FileExt,
		"file_size":      info.FileSize,
		"path":           info.Path,
		"last_access":    info.LastAccess,
		"status":         "pending",
		"source_type":    "context_menu",
		"is_favorite":    false,
		"category_id":    cfg.DefaultCategoryID,
		"subcategory_id": cfg.DefaultSubcategoryID,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error serializando: %w", err)
	}

	resp, err := http.Post(
		"http://127.0.0.1:8090/api/collections/documents/records",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("error conectando al servidor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errBody struct {
			Message string `json:"message"`
			Data    any    `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("servidor respondió %d: %s — %v", resp.StatusCode, errBody.Message, errBody.Data)
	}

	log.Printf("documento agregado: %s (%s)", info.FileName, info.Path)
	return nil
}

func AddFileViaDAO(app *pocketbase.PocketBase, info *addfile.Info) error {
	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		return fmt.Errorf("no se encontró la colección documents: %w", err)
	}

	catRecords, err := app.FindRecordsByFilter("categories",
		"name = {:name}", "", 1, 0,
		map[string]any{"name": "Sin categorizar"})
	if err != nil || len(catRecords) == 0 {
		return fmt.Errorf("categoría 'Sin categorizar' no encontrada")
	}
	catID := catRecords[0].Id

	subRecords, err := app.FindRecordsByFilter("subcategories",
		"name = {:name} && category_id = {:cat_id}", "", 1, 0,
		map[string]any{"name": "General", "cat_id": catID})
	if err != nil || len(subRecords) == 0 {
		return fmt.Errorf("subcategoría 'General' no encontrada")
	}
	subID := subRecords[0].Id

	rec := core.NewRecord(documents)
	rec.Set("name", info.Name)
	rec.Set("file_name", info.FileName)
	rec.Set("file_ext", info.FileExt)
	rec.Set("file_size", info.FileSize)
	rec.Set("path", info.Path)
	rec.Set("last_access", info.LastAccess)
	rec.Set("status", "pending")
	rec.Set("source_type", "context_menu")
	rec.Set("is_favorite", false)
	rec.Set("category_id", catID)
	rec.Set("subcategory_id", subID)

	if err := app.Save(rec); err != nil {
		return fmt.Errorf("error guardando documento: %w", err)
	}

	log.Printf("documento agregado: %s (%s)", info.FileName, info.Path)
	return nil
}
