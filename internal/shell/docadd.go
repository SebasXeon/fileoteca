package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"SebasXeon/Fileoteca/internal/addfile"
)

func AddFileViaHTTP(info *addfile.Info) error {
	body := map[string]any{
		"name":        info.Name,
		"file_name":   info.FileName,
		"file_ext":    info.FileExt,
		"file_size":   info.FileSize,
		"path":        info.Path,
		"last_access": info.LastAccess,
		"status":      "pending",
		"source_type": "context_menu",
		"is_favorite": false,
	}

	catID, err := resolveCategoryID()
	if err != nil {
		return fmt.Errorf("no se pudo resolver categoría por defecto: %w", err)
	}
	subID, err := resolveSubcategoryID(catID)
	if err != nil {
		return fmt.Errorf("no se pudo resolver subcategoría por defecto: %w", err)
	}
	body["category_id"] = catID
	body["subcategory_id"] = subID

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error serializando datos: %w", err)
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
		return fmt.Errorf("servidor respondió con error %d", resp.StatusCode)
	}

	log.Printf("documento agregado: %s (%s)", info.FileName, info.Path)
	return nil
}

func resolveCategoryID() (string, error) {
	resp, err := http.Get(
		"http://127.0.0.1:8090/api/collections/categories/records?filter=(name='Sin categorizar')&fields=id&perPage=1",
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Items) == 0 {
		return "", fmt.Errorf("categoría 'Sin categorizar' no encontrada")
	}
	return result.Items[0].ID, nil
}

func resolveSubcategoryID(catID string) (string, error) {
	url := fmt.Sprintf(
		"http://127.0.0.1:8090/api/collections/subcategories/records?filter=(name='General'&&category_id='%s')&fields=id&perPage=1",
		catID,
	)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Items) == 0 {
		return "", fmt.Errorf("subcategoría 'General' no encontrada para categoría %s", catID)
	}
	return result.Items[0].ID, nil
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
