package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ContextMenuRegistered bool   `json:"context_menu_registered"`
	InstalledPath         string `json:"installed_path"`
	DefaultCategoryID     string `json:"default_category_id"`
	DefaultSubcategoryID  string `json:"default_subcategory_id"`
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("no se pudo obtener APPDATA: %w", err)
	}
	return filepath.Join(dir, "Fileoteca"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	p, err := configPath()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("error leyendo config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, fmt.Errorf("error parseando config: %w", err)
	}
	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("error creando directorio config: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando config: %w", err)
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		return fmt.Errorf("error escribiendo config: %w", err)
	}
	return nil
}
