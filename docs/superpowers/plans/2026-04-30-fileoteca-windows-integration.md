# Windows Context Menu & System Tray — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Windows shell integration: context menu "Agregar a Fileoteca" for 23 document extensions, system tray with quick access to web UI, and single-binary mode dispatch via `--add` flag.

**Architecture:** Two new internal packages — `addfile` for file validation/metadata extraction and `shell` for all Windows-specific integration (registry, config, named mutex, tray). `main.go` dispatches between normal/server mode and context-menu mode early in startup, stripping custom flags before PocketBase's cobra CLI sees them.

**Tech Stack:** Go 1.25, PocketBase v0.36.9, `github.com/getlantern/systray`, `golang.org/x/sys/windows`

---

## File Structure

| File | Purpose |
|------|---------|
| `main.go` | Rewrite: mode dispatch, orchestration, shutdown coordination |
| `internal/addfile/addfile.go` | File validation, supported extensions list, metadata extraction |
| `internal/shell/config.go` | Config struct, load/save `%APPDATA%\Fileoteca\config.json` |
| `internal/shell/registry.go` | Windows registry: install/uninstall context menu per extension |
| `internal/shell/tray.go` | System tray: icon generation, menu setup, browser opening |
| `internal/shell/server.go` | Server init, named mutex, default category creation, lifecycle |
| `internal/shell/docadd.go` | Add document via HTTP API or direct PocketBase DAO |
| `go.mod` | Add `systray` and `x/sys/windows` dependencies |

---

### Task 1: Add Go Dependencies

**Files:** Modify `go.mod`

- [ ] **Step 1: Run `go get` to add dependencies**

```bash
go get github.com/getlantern/systray@latest
go get golang.org/x/sys@latest
```

- [ ] **Step 2: Run `go mod tidy`**

```bash
go mod tidy
```

- [ ] **Step 3: Verify `go build` compiles (will fail — expected until we write the code)**

```bash
go build -o Fileoteca.exe
```

Expected: compilation succeeds (after tasks 2-5 are done). If it fails now with unused dependency errors, that's fine — we'll use them in later tasks.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add systray and x/sys for Windows shell integration"
```

---

### Task 2: Create `internal/addfile/addfile.go` — File Validation & Metadata

**Files:** Create `internal/addfile/addfile.go`

- [ ] **Step 1: Write the file**

```go
package addfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SupportedExt maps lowercase extensions (with dot) to true for the context menu.
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

// Info holds the metadata extracted from a file path.
type Info struct {
	Name       string `json:"name"`
	FileName   string `json:"file_name"`
	FileExt    string `json:"file_ext"`
	FileSize   int64  `json:"file_size"`
	Path       string `json:"path"`
	LastAccess string `json:"last_access"`
}

// Validate checks that the file exists and has a supported extension.
// Returns an error if the file is missing or the extension is unsupported.
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

// Extract returns metadata about the file at the given path.
// The extension is stored without the leading dot to match the DB convention.
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/addfile/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/addfile/addfile.go
git commit -m "feat: add file validation and metadata extraction for context menu"
```

---

### Task 3: Create `internal/shell/config.go` — App Config

**Files:** Create `internal/shell/config.go`

- [ ] **Step 1: Write the file**

```go
package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds persistent app configuration stored in %APPDATA%\Fileoteca\config.json
type Config struct {
	ContextMenuRegistered bool   `json:"context_menu_registered"`
	InstalledPath         string `json:"installed_path"`
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

// LoadConfig reads the config file. Returns defaults if the file doesn't exist.
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

// SaveConfig writes the config file, creating directories as needed.
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/shell/...
```

Expected: fails because `shell` package is empty of other files — but this file individually is valid. We'll compile after adding all files.

- [ ] **Step 3: Commit**

```bash
git add internal/shell/config.go
git commit -m "feat: add config file management for app settings"
```

---

### Task 4: Create `internal/shell/registry.go` — Windows Context Menu

**Files:** Create `internal/shell/registry.go`

- [ ] **Step 1: Write the file**

```go
package shell

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Extensions registered in the Windows context menu.
var menuExtensions = []string{
	".pdf",
	".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
	".txt", ".csv", ".rtf", ".md", ".html", ".htm", ".xml", ".json",
	".odt", ".ods", ".odp",
	".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".webp", ".tiff", ".ico",
}

const menuLabel = "Agregar a Fileoteca"
const shellKeyName = "Fileoteca"

// installOneExt writes a single extension's shell key to the registry.
// Returns whether any key was actually created (vs already existing).
func installOneExt(k registry.Key, ext, exePath string) (bool, error) {
	shellPath := fmt.Sprintf(`SystemFileAssociations\%s\shell\%s`, ext, shellKeyName)
	cmdPath := shellPath + `\command`

	key, _, err := registry.CreateKey(k, shellPath, registry.SET_VALUE)
	if err != nil {
		return false, fmt.Errorf("error creando clave para %s: %w", ext, err)
	}
	defer key.Close()

	existing, _, err := key.GetStringValue("")
	isNew := err != nil || existing != menuLabel

	if err := key.SetStringValue("", menuLabel); err != nil {
		return false, err
	}
	if err := key.SetStringValue("Icon", exePath+",0"); err != nil {
		return false, err
	}

	cmdKey, _, err := registry.CreateKey(k, cmdPath, registry.SET_VALUE)
	if err != nil {
		return false, fmt.Errorf("error creando comando para %s: %w", ext, err)
	}
	defer cmdKey.Close()

	command := fmt.Sprintf(`"%s" --add "%%1"`, exePath)
	if err := cmdKey.SetStringValue("", command); err != nil {
		return false, err
	}

	return isNew, nil
}

// IsRegistered checks whether the context menu is already installed
// by testing the first extension's registry key.
func IsRegistered() (bool, error) {
	k, err := registry.OpenKey(registry.CLASSES_ROOT,
		fmt.Sprintf(`SystemFileAssociations\%s\shell\%s`, menuExtensions[0], shellKeyName),
		registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer k.Close()
	return true, nil
}

// Install creates context menu entries for all supported extensions.
// Returns an error if none could be created.
func Install() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("no se pudo obtener ruta del ejecutable: %w", err)
	}

	var errs []string
	installed := 0
	for _, ext := range menuExtensions {
		_, err := installOneExt(registry.CLASSES_ROOT, ext, exePath)
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

// EnsureContextMenu checks whether the menu is installed and up-to-date.
// Returns true if installation was performed.
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/shell/...
```

Expected: passes now that we have both config.go and registry.go.

- [ ] **Step 3: Commit**

```bash
git add internal/shell/registry.go
git commit -m "feat: add Windows context menu registry integration"
```

---

### Task 5: Create `internal/shell/tray.go` — System Tray

**Files:** Create `internal/shell/tray.go`, modify `internal/shell/server.go` (not yet created — created in Task 6)

The tray starts after PocketBase is ready. It runs on the main goroutine since `systray.Run` blocks. We need a reference to the PocketBase app to stop it on exit.

The tray needs an icon. We generate a minimal 16x16 32bpp .ico in-memory (a colored square with the app's accent color).

- [ ] **Step 1: Write the file**

```go
package shell

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"

	"github.com/getlantern/systray"
)

const serverURL = "http://127.0.0.1:8090"
const settingsURL = serverURL + "/settings"

// generateIcon creates a minimal 16x16 32bpp .ico (blue-ish square) in-memory.
// This avoids needing an external .ico file; users can replace with a custom icon later.
func generateIcon() []byte {
	const size = 16
	const bpp = 32

	// XOR mask: bottom-up BGRA pixels
	xorSize := size * size * 4
	// AND mask: 1bpp, 4-byte-aligned per row
	andRowBytes := ((size + 31) / 32) * 4
	andSize := size * andRowBytes

	bmpDataSize := 40 + xorSize + andSize
	icoDataSize := 6 + 16 + bmpDataSize

	buf := make([]byte, icoDataSize)
	pos := 0

	// ICO header
	buf[pos] = 0; pos++  // reserved
	buf[pos] = 0; pos++
	buf[pos] = 1; pos++  // type: ICO
	buf[pos] = 0; pos++
	buf[pos] = 1; pos++  // count: 1
	buf[pos] = 0; pos++

	// Directory entry
	buf[pos] = size; pos++  // width (0 = 256)
	buf[pos] = size; pos++  // height (0 = 256)
	buf[pos] = 0; pos++     // palette
	buf[pos] = 0; pos++     // reserved
	binary.LittleEndian.PutUint16(buf[pos:], 1); pos += 2  // planes
	binary.LittleEndian.PutUint16(buf[pos:], uint16(bpp)); pos += 2 // bpp
	binary.LittleEndian.PutUint32(buf[pos:], uint32(bmpDataSize)); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], uint32(6+16)); pos += 4 // offset

	// BITMAPINFOHEADER
	binary.LittleEndian.PutUint32(buf[pos:], 40); pos += 4        // biSize
	binary.LittleEndian.PutUint32(buf[pos:], uint32(size)); pos += 4  // biWidth
	binary.LittleEndian.PutUint32(buf[pos:], uint32(size*2)); pos += 4 // biHeight (XOR + AND)
	binary.LittleEndian.PutUint16(buf[pos:], 1); pos += 2         // biPlanes
	binary.LittleEndian.PutUint16(buf[pos:], uint16(bpp)); pos += 2 // biBitCount
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4         // biCompression
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4         // biSizeImage
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4         // biXPelsPerMeter
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4         // biYPelsPerMeter
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4         // biClrUsed
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4         // biClrImportant

	// XOR mask: fill with a solid color (bottom row first)
	color := [4]byte{0x20, 0x80, 0xFF, 0xFF} // BGRA: blue accent
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			copy(buf[pos:], color[:])
			pos += 4
		}
	}

	// AND mask: all zeros = fully opaque
	for i := 0; i < andSize; i++ {
		buf[pos] = 0
		pos++
	}

	return buf
}

// openBrowser opens the given URL in the default browser on Windows.
func openBrowser(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

// StartTray sets up the system tray and blocks until the user quits.
// The onReady callback receives a function to stop the server.
// This MUST be called from the main goroutine.
func StartTray(stopServerFn func()) {
	onReady := func() {
		systray.SetIcon(generateIcon())
		systray.SetTooltip("Fileoteca")

		// Left-click: open app
		mOpen := systray.AddMenuItem("Abrir Fileoteca", "Abrir en el navegador")
		go func() {
			for range mOpen.ClickedCh {
				if err := openBrowser(serverURL); err != nil {
					fmt.Fprintf(os.Stderr, "error abriendo navegador: %v\n", err)
				}
			}
		}()

		systray.AddSeparator()

		// Right-click menu items
		mConfig := systray.AddMenuItem("Configurar", "Abrir configuración")
		go func() {
			for range mConfig.ClickedCh {
				if err := openBrowser(settingsURL); err != nil {
					fmt.Fprintf(os.Stderr, "error abriendo configuración: %v\n", err)
				}
			}
		}()

		systray.AddSeparator()

		mQuit := systray.AddMenuItem("Cerrar", "Cerrar Fileoteca")
		go func() {
			<-mQuit.ClickedCh
			stopServerFn()
			systray.Quit()
		}()
	}

	onExit := func() {
		// Cleanup handled by stopServerFn already
	}

	systray.Run(onReady, onExit)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/shell/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/shell/tray.go
git commit -m "feat: add system tray with icon, open browser, and quit support"
```

---

### Task 6: Create `internal/shell/server.go` — Server Lifecycle

**Files:** Create `internal/shell/server.go`, modify `internal/shell/tray.go`

This file handles the server startup logic shared between normal mode and context-menu (cold start) mode: named mutex creation, default category bootstrap, and the server start goroutine.

PocketBase needs to run its `serve` command. We start it in a goroutine and wait for bootstrap (migrations + DB ready) via a channel signal. After that, we can use `app.DB()` directly.

- [ ] **Step 1: Write the file**

```go
package shell

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"unsafe"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/sys/windows"
)

var (
	mutexHandle windows.Handle
	mutexOnce   sync.Once
)

// createMutex creates a named mutex to prevent duplicate instances.
// Returns an error if the mutex already exists (app already running).
func createMutex() error {
	name, _ := syscall.UTF16PtrFromString("FileotecaServer")
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		return fmt.Errorf("error creando mutex: %w", err)
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(handle)
		return fmt.Errorf("otra instancia de Fileoteca ya está corriendo")
	}
	mutexHandle = handle
	return nil
}

// releaseMutex closes the named mutex handle.
func releaseMutex() {
	if mutexHandle != 0 {
		windows.CloseHandle(mutexHandle)
		mutexHandle = 0
	}
}

// IsServerRunning checks whether the named mutex exists (server is running).
func IsServerRunning() bool {
	name, _ := syscall.UTF16PtrFromString("FileotecaServer")
	handle, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return false
	}
	windows.CloseHandle(handle)
	return true
}

// ensureDefaultCategory creates the "Sin categorizar" category and "General"
// subcategory if they don't already exist. Uses app.DB() directly.
func ensureDefaultCategory(app *pocketbase.PocketBase) error {
	// Find or create category "Sin categorizar"
	categories, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		return fmt.Errorf("no se encontró la colección categories: %w", err)
	}

	subcategories, err := app.FindCollectionByNameOrId("subcategories")
	if err != nil {
		return fmt.Errorf("no se encontró la colección subcategories: %w", err)
	}

	// Check if "Sin categorizar" category exists
	catRecords, err := app.FindRecordsByFilter("categories", "name = {:name}", "", -1, 0,
		map[string]any{"name": "Sin categorizar"})
	if err != nil {
		return fmt.Errorf("error buscando categoría por defecto: %w", err)
	}

	var catID string
	if len(catRecords) == 0 {
		rec := core.NewRecord(categories)
		rec.Set("name", "Sin categorizar")
		rec.Set("description", "Categoría por defecto para archivos agregados")
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("error creando categoría por defecto: %w", err)
		}
		catID = rec.Id
	} else {
		catID = catRecords[0].Id
	}

	// Check if "General" subcategory exists under this category
	subRecords, err := app.FindRecordsByFilter("subcategories",
		"name = {:name} && category_id = {:cat_id}", "", -1, 0,
		map[string]any{"name": "General", "cat_id": catID})
	if err != nil {
		return fmt.Errorf("error buscando subcategoría por defecto: %w", err)
	}

	if len(subRecords) == 0 {
		rec := core.NewRecord(subcategories)
		rec.Set("name", "General")
		rec.Set("category_id", catID)
		rec.Set("model_name", "general")
		rec.Set("is_default", true)
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("error creando subcategoría por defecto: %w", err)
		}
	}

	return nil
}

// StartServer creates the PocketBase app, ensures config and categories,
// and starts the HTTP server in a goroutine. Returns the app instance
// and a stop function for cleanup.
func StartServer() (*pocketbase.PocketBase, func(), error) {
	if err := createMutex(); err != nil {
		return nil, nil, err
	}

	// Ensure context menu
	installed, err := EnsureContextMenu()
	if err != nil {
		log.Printf("aviso: no se pudo registrar el menú contextual: %v", err)
	} else if installed {
		log.Println("menú contextual registrado correctamente")
	}

	app := pocketbase.New()

	// Register migrations from the embedded migrations package
	// (PocketBase auto-discovers them via init() in the migrations package)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/{path...}", apis.Static(os.DirFS("./pb_public"), false))
		return se.Next()
	})

	// Signal when bootstrap is complete (migrations + DB ready)
	bootstrapDone := make(chan struct{})
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		close(bootstrapDone)
		return e.Next()
	})

	// Start server in goroutine
	go func() {
		if err := app.Start(); err != nil {
			log.Fatalf("error iniciando servidor: %v", err)
		}
	}()

	// Wait for bootstrap to complete (DB + migrations ready)
	<-bootstrapDone

	// Ensure the default category exists (requires DB to be ready)
	if err := ensureDefaultCategory(app); err != nil {
		log.Printf("aviso: no se pudo crear categoría por defecto: %v", err)
	}

	stopFn := func() {
		log.Println("deteniendo servidor...")
		_ = app.Stop()
		releaseMutex()
	}

	return app, stopFn, nil
}

// This references imports needed in server.go that we need to add.
// The actual import block is at the top — verify these are present:
//   "github.com/pocketbase/pocketbase"
//   "github.com/pocketbase/pocketbase/core"
//   "github.com/pocketbase/pocketbase/plugins/migratecmd"
//   "github.com/pocketbase/pocketbase/apis"
//   "github.com/pocketbase/pocketbase/tools/osutils"
```

Wait — this imports `pocketbase` and `core`, making `internal/shell` depend on PocketBase. That's fine since `main.go` already depends on both. But we also need the `migratecmd` and `apis` and `osutils` imports. Let me fix the import block.

- [ ] **Step 2: Fix the file — correct imports**

The file needs these imports at the top:

```go
package shell

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"golang.org/x/sys/windows"
)
```

Edit `internal/shell/server.go` to use this import block. Remove the comment at the bottom.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/shell/...
```

Expected: compiles successfully.

- [ ] **Step 4: Commit**

```bash
git add internal/shell/server.go
git commit -m "feat: add server lifecycle with named mutex and default category bootstrap"
```

---

### Task 7: Create `internal/shell/docadd.go` — Add Document via API or DAO

**Files:** Create `internal/shell/docadd.go`

- [ ] **Step 1: Write the file**

```go
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

// AddFileViaHTTP sends a POST to the running PocketBase API to add a document.
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

	// Resolve default category and subcategory IDs via API
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

// resolveCategoryID fetches the "Sin categorizar" category ID via the API.
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

// resolveSubcategoryID fetches the "General" subcategory ID for a category via the API.
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

// AddFileViaDAO inserts a document record directly into the database.
// Used when the server is starting fresh (cold start from context menu).
func AddFileViaDAO(app *pocketbase.PocketBase, info *addfile.Info) error {
	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		return fmt.Errorf("no se encontró la colección documents: %w", err)
	}

	// Find default category
	catRecords, err := app.FindRecordsByFilter("categories",
		"name = {:name}", "", 1, 0,
		map[string]any{"name": "Sin categorizar"})
	if err != nil || len(catRecords) == 0 {
		return fmt.Errorf("categoría 'Sin categorizar' no encontrada")
	}
	catID := catRecords[0].Id

	// Find default subcategory
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
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/shell/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/shell/docadd.go
git commit -m "feat: add document via HTTP API or direct DAO from context menu"
```

---

### Task 8: Rewrite `main.go` — Mode Dispatch & Orchestration

**Files:** Rewrite `main.go`

This is the entry point. It inspects `os.Args` before PocketBase sees them, dispatches to the right mode, and coordinates server + tray lifecycle.

- [ ] **Step 1: Write the new `main.go`**

```go
package main

import (
	"fmt"
	"log"
	"os"

	"SebasXeon/Fileoteca/internal/addfile"
	"SebasXeon/Fileoteca/internal/shell"

	_ "SebasXeon/Fileoteca/migrations"
)

func main() {
	var addFilePath string

	// Check for --add flag before PocketBase cobra sees it
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--add" && i+1 < len(os.Args) {
			addFilePath = os.Args[i+1]
			break
		}
	}

	// Context menu mode with file path
	if addFilePath != "" {
		info, err := addfile.Extract(addFilePath)
		if err != nil {
			log.Printf("error: %v\n", err)
			os.Exit(1)
		}

		// Check if server is already running
		if shell.IsServerRunning() {
			if err := shell.AddFileViaHTTP(info); err != nil {
				log.Printf("error agregando documento: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Cold start: strip --add flag and file path, then start normally
		// Remove the --add flag and its argument from os.Args
		newArgs := os.Args[:1] // keep just the exe name
		for i := 1; i < len(os.Args); i++ {
			if os.Args[i] == "--add" {
				i++ // skip the file path argument too
				continue
			}
			newArgs = append(newArgs, os.Args[i])
		}
		os.Args = newArgs

		// Start server, then add the file
		app, stopFn, err := shell.StartServer()
		if err != nil {
			log.Fatalf("error iniciando servidor: %v\n", err)
		}

		if err := shell.AddFileViaDAO(app, info); err != nil {
			log.Printf("error agregando documento vía DAO: %v\n", err)
		}

		// Run tray (blocks until quit)
		shell.StartTray(stopFn)
		return
	}

	// Normal mode: start server + tray
	_, stopFn, err := shell.StartServer()
	if err != nil {
		log.Fatalf("error iniciando servidor: %v\n", err)
	}

	fmt.Println("Fileoteca iniciada. Haz clic en el icono del área de notificación.")
	shell.StartTray(stopFn)
}
```

- [ ] **Step 2: Build the binary**

```bash
go build -o Fileoteca.exe
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add mode dispatch, context menu handling, and tray orchestration"
```

---

### Task 9: Build and Verify

**Files:** None (build verification)

- [ ] **Step 1: Full build**

```bash
go build -o Fileoteca.exe
```

Expected: successful compilation, no warnings.

- [ ] **Step 2: Verify binary runs (normal mode — will need to kill with Ctrl+C or tray icon)**

Run the binary briefly to verify it starts:

```bash
.\Fileoteca.exe
```

Expected: should print "Fileoteca iniciada." and show a tray icon. The server starts on port 8090. Press Ctrl+C to stop.

Note: The tray icon won't appear in VSCode terminal context. Run from a normal PowerShell window or File Explorer.

- [ ] **Step 3: Verify context menu registration**

After running once, check the registry:

```powershell
Get-Item "Registry::HKEY_CLASSES_ROOT\SystemFileAssociations\.pdf\shell\Fileoteca"
```

Expected: shows the "Agregar a Fileoteca" menu entry.

- [ ] **Step 4: Verify `--add` flag parsing**

```bash
.\Fileoteca.exe --add "C:\nonexistent\file.pdf"
```

Expected: prints error "archivo no encontrado" and exits 1.

- [ ] **Step 5: Verify extension validation**

```bash
.\Fileoteca.exe --add "C:\Windows\System32\cmd.exe"
```

Expected: prints error "extensión no soportada: .exe" and exits 1.

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "chore: verify build and add any remaining files"
```

---

## Testing Checklist

After all tasks are complete, run these manual tests:

- [ ] Normal launch: tray icon appears, server runs on port 8090
- [ ] Tray left-click: opens `http://127.0.0.1:8090/` in browser
- [ ] Tray right-click → Configurar: opens `http://127.0.0.1:8090/settings`
- [ ] Tray right-click → Cerrar: server stops, tray disappears
- [ ] Context menu: right-click a `.pdf` in File Explorer, select "Agregar a Fileoteca"
- [ ] Context menu (server running): file appears in the web UI
- [ ] Context menu (server not running): server starts, tray appears, file is added
- [ ] Config file: `%APPDATA%\Fileoteca\config.json` exists after first run
- [ ] Duplicate instance: running `Fileoteca.exe` twice prevents second instance
- [ ] Unsupported file: context menu on `.exe` should not show the menu entry
