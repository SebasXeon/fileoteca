package addfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var SupportedExt = map[string]bool{
	".pdf": true,
	".doc": true, ".docx": true,
	".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true,
	".txt": true, ".csv": true, ".rtf": true, ".md": true,
	".html": true, ".htm": true, ".xml": true, ".json": true,
	".odt": true, ".ods": true, ".odp": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".svg": true, ".webp": true, ".tiff": true, ".ico": true,
}

type Info struct {
	Name       string `json:"name"`
	FileName   string `json:"file_name"`
	FileExt    string `json:"file_ext"`
	FileSize   int64  `json:"file_size"`
	Path       string `json:"path"`
	LastAccess string `json:"last_access"`
}

func Validate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("archivo no encontrado: %s", path)
	}
	if info.IsDir() {
		return fmt.Errorf("no se puede agregar un directorio: %s", path)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if !SupportedExt[ext] {
		return fmt.Errorf("extensión no soportada: %s", ext)
	}
	return nil
}

func Extract(path string) (*Info, error) {
	if err := Validate(path); err != nil {
		return nil, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el archivo: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	name := strings.TrimSuffix(filepath.Base(path), ext)
	if name == "" {
		name = filepath.Base(path)
	}
	return &Info{
		Name:       name,
		FileName:   filepath.Base(path),
		FileExt:    strings.TrimPrefix(ext, "."),
		FileSize:   stat.Size(),
		Path:       path,
		LastAccess: stat.ModTime().Format(time.RFC3339),
	}, nil
}
