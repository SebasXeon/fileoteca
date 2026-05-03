package api

import (
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

var inlineTypes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true,
	"image/bmp": true, "image/svg+xml": true, "image/webp": true,
	"image/tiff": true, "image/x-icon": true, "image/vnd.microsoft.icon": true,
	"application/pdf": true,
	"text/plain": true, "text/csv": true, "text/markdown": true,
	"text/html": true, "application/json": true, "application/xml": true,
	"text/xml": true, "text/rtf": true,
}

func OpenDocumentHandler(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "id requerido"})
		}

		record, err := app.FindRecordById("documents", id)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "documento no encontrado"})
		}

		filePath := record.GetString("path")
		fileName := record.GetString("file_name")
		fileExt := strings.ToLower(record.GetString("file_ext"))

		var resolvedPath string

		// 1. Intentar usar path local
		if filePath != "" {
			if _, err := os.Stat(filePath); err == nil {
				resolvedPath = filePath
			}
		}

		// 2. Fallback a archivo de PocketBase storage
		if resolvedPath == "" {
			fileField := record.GetString("file")
			if fileField != "" {
				storagePath := filepath.Join(app.DataDir(), "storage", record.BaseFilesPath(), fileField)
				if _, err := os.Stat(storagePath); err == nil {
					resolvedPath = storagePath
				}
			}
		}

		if resolvedPath == "" {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "archivo no encontrado"})
		}

		mimeType := mime.TypeByExtension("." + fileExt)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Si es visualizable en navegador, servir inline
		if inlineTypes[mimeType] {
			e.Response.Header().Del("X-Frame-Options")
			e.Response.Header().Set("Content-Type", mimeType)
			e.Response.Header().Set("Content-Disposition", "inline; filename=\""+fileName+"\"")

			f, err := os.Open(resolvedPath)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "no se pudo leer el archivo"})
			}
			defer f.Close()

			stat, err := f.Stat()
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "no se pudo leer el archivo"})
			}

			http.ServeContent(e.Response, e.Request, fileName, stat.ModTime(), f)
			return nil
		}

		// Abrir externamente con app por defecto en Windows
		cmd := exec.Command("cmd", "/c", "start", "", resolvedPath)
		if err := cmd.Start(); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "no se pudo abrir el archivo: " + err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"action": "opened_externally"})
	}
}
