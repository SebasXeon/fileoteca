package shell

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

var menuExtensions = []string{
	".pdf",
	".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
	".txt", ".csv", ".rtf", ".md", ".html", ".htm", ".xml", ".json",
	".odt", ".ods", ".odp",
	".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".webp", ".tiff", ".ico",
}

const menuLabel = "Agregar a Fileoteca"
const shellKeyName = "Fileoteca"

// regBase is the registry root for per-user context menu entries.
// HKCU\Software\Classes merges into HKCR without requiring admin rights.
const regBase = `Software\Classes`

func installOneExt(ext, exePath string) error {
	shellPath := fmt.Sprintf(`%s\SystemFileAssociations\%s\shell\%s`, regBase, ext, shellKeyName)
	cmdPath := shellPath + `\command`

	key, _, err := registry.CreateKey(registry.CURRENT_USER, shellPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("error creando clave para %s: %w", ext, err)
	}
	defer key.Close()

	if err := key.SetStringValue("", menuLabel); err != nil {
		return err
	}
	if err := key.SetStringValue("Icon", exePath+",0"); err != nil {
		return err
	}

	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, cmdPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("error creando comando para %s: %w", ext, err)
	}
	defer cmdKey.Close()

	command := fmt.Sprintf(`"%s" --add "%%1"`, exePath)
	if err := cmdKey.SetStringValue("", command); err != nil {
		return err
	}

	return nil
}

func IsRegistered() (bool, error) {
	checkPath := fmt.Sprintf(`%s\SystemFileAssociations\%s\shell\%s`, regBase, menuExtensions[0], shellKeyName)
	k, err := registry.OpenKey(registry.CURRENT_USER, checkPath, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer k.Close()
	return true, nil
}

func Install() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("no se pudo obtener ruta del ejecutable: %w", err)
	}

	var errs []string
	installed := 0
	for _, ext := range menuExtensions {
		err := installOneExt(ext, exePath)
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			installed++
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errores instalando menú contextual (%d/%d): %s",
			installed, len(menuExtensions), strings.Join(errs, "; "))
	}

	cfg, _ := LoadConfig()
	cfg.ContextMenuRegistered = true
	cfg.InstalledPath = exePath
	_ = SaveConfig(cfg)

	return nil
}

func EnsureContextMenu() (bool, error) {
	exePath, err := os.Executable()
	if err != nil {
		return false, err
	}

	cfg, err := LoadConfig()
	if err != nil {
		return false, err
	}

	if cfg.ContextMenuRegistered && cfg.InstalledPath == exePath {
		registered, err := IsRegistered()
		if err == nil && registered {
			return false, nil
		}
	}

	if err := Install(); err != nil {
		return false, err
	}
	return true, nil
}
