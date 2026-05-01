# Windows Context Menu & System Tray Integration

**Date:** 2026-04-30
**Status:** approved

## Overview

Add Windows shell integration to `Fileoteca.exe` so users can:
- Right-click documents in File Explorer → "Agregar a Fileoteca" to add them by reference
- Run the app as a system tray application with quick access to the web UI and settings

## Architecture

### Single binary, three execution modes

`Fileoteca.exe` inspects `os.Args` before PocketBase's cobra CLI sees them:

| Mode | Trigger | Behavior |
|------|---------|----------|
| **Normal** | No args | Register context menu (if needed) → start PocketBase server → show tray icon |
| **Context menu (server running)** | `--add <path>`, named mutex exists | POST to local PocketBase API → exit 0 |
| **Context menu (server not running)** | `--add <path>`, named mutex absent | Start PocketBase → insert document via internal DAO → show tray icon → keep running |

The `--add` flag is intercepted early in `main()`. In the "server running" case, PocketBase is never initialized — just an HTTP POST and exit. In the "server not running" case, the full app lifecycle kicks in.

### Instance detection

A Windows named mutex (`FileotecaServer`) is created at startup. Context menu invocations check for its existence via `CreateMutex` / `OpenMutex`.

- Mutex exists → server running → HTTP POST and exit
- Mutex absent → start full app

## System Tray

Library: `github.com/getlantern/systray`

- **Icon:** Embedded via `embed.FS` as a `.ico` file in the binary
- **Left-click:** Opens `http://127.0.0.1:8090/` in the default browser (`rundll32 url.dll,FileProtocolHandler`)
- **Right-click menu:**
  - **Configurar** → Opens `http://127.0.0.1:8090/settings` in the browser
  - **Cerrar** → Stops PocketBase server → quits tray → exits process

The tray runs on the main goroutine (`systray.Run` blocks). PocketBase runs in a separate goroutine. On "Cerrar", PocketBase is stopped first via `app.Stop()`, then `systray.Quit()` is called.

## Windows Context Menu

### Registry location

Per-extension registration under `HKEY_CLASSES_ROOT\SystemFileAssociations`:

```
SystemFileAssociations\.pdf\shell\Fileoteca
    (Default) = "Agregar a Fileoteca"
    Icon = "<installed_path>,0"

SystemFileAssociations\.pdf\shell\Fileoteca\command
    (Default) = "<installed_path>" --add "%1"
```

This is repeated for every supported extension.

### Supported extensions

PDF, Office documents, text files, and images:

| Category | Extensions |
|----------|-----------|
| PDF | `.pdf` |
| Office | `.doc`, `.docx`, `.xls`, `.xlsx`, `.ppt`, `.pptx` |
| Text | `.txt`, `.csv`, `.rtf`, `.md`, `.html`, `.htm`, `.xml`, `.json` |
| OpenDocument | `.odt`, `.ods`, `.odp` |
| Images | `.png`, `.jpg`, `.jpeg`, `.gif`, `.bmp`, `.svg`, `.webp`, `.tiff`, `.ico` |

Total: 23 extensions.

### Config file

Location: `%APPDATA%\Fileoteca\config.json`

```json
{
  "context_menu_registered": true,
  "installed_path": "C:\\Users\\...\\Fileoteca.exe"
}
```

At startup in normal mode, the config is read. If `context_menu_registered` is `false` or `installed_path` differs from the current executable path, registry keys are written/updated. Otherwise, registration is skipped.

### Uninstallation

Not automatic on exit. The context menu persists across restarts. A future "Desinstalar menú contextual" option could be added to the tray menu.

## Document Addition Logic

When `--add <path>` is invoked:

1. Validate file exists and has a supported extension
2. Extract metadata from `os.Stat`: filename, extension, size, mod time, absolute path
3. Map to the `documents` collection fields:

| Source | DB field | Notes |
|--------|----------|-------|
| File path stem | `name` | Required, presentable — the document display name |
| Full filename | `file_name` | Required — e.g. `"report.pdf"` |
| Extension | `file_ext` | Required — e.g. `"pdf"` (no leading dot) |
| `os.Stat().Size()` | `file_size` | Integer bytes |
| Absolute path | `path` | The original file location |
| `os.Stat().ModTime()` | `last_access` | Date field |
| Fixed | `status` | `"pending"` — initial status |
| Fixed | `source_type` | `"context_menu"` |
| Flag | `is_favorite` | `false` |

4. `category_id` and `subcategory_id` are **required** relation fields. On normal startup, the app ensures a default "Sin categorizar" category exists with a "General" subcategory (creating them if absent). Context menu additions always use this default pair.

**If server is running:** `POST http://127.0.0.1:8090/api/collections/documents/records`

```json
{
  "name": "report",
  "file_name": "report.pdf",
  "file_ext": "pdf",
  "file_size": 204800,
  "path": "C:\\Users\\Sebas\\Documents\\report.pdf",
  "last_access": "2026-04-30T19:30:00.000Z",
  "status": "pending",
  "source_type": "context_menu",
  "is_favorite": false,
  "category_id": "<id of default category>",
  "subcategory_id": "<id of default subcategory>"
}
```

**If server is NOT running:** Insert via PocketBase's internal `app.DB().NewRecord()` and `app.Save()` directly in Go.

### User feedback

Log messages to stdout (console). When invoked from the context menu, the console is not visible; feedback is implicit via the tray icon appearing (for the cold-start case) or the file appearing in the web UI.

## Dependencies

New Go modules to add to `go.mod`:

- `github.com/getlantern/systray` — system tray
- `golang.org/x/sys` — Windows API bindings (registry, mutex)

Existing dependencies (no changes):
- `github.com/pocketbase/pocketbase` v0.36.9
- `modernc.org/sqlite` (indirect, via PocketBase)

## File Changes Summary

| File | Change |
|------|--------|
| `main.go` | Rewrite: mode dispatch, tray setup, context menu registration, document addition |
| `go.mod` | Add `systray` and `x/sys` dependencies |
| `assets/tray.ico` | New: embedded tray icon |
| `internal/registry/registry.go` | New: Windows registry helpers |
| `internal/contextmenu/contextmenu.go` | New: context menu install/uninstall logic |
| `internal/tray/tray.go` | New: tray setup and event handling |
| `internal/addfile/addfile.go` | New: file validation and document creation logic |

## Error Handling

- **Registry write failure:** Log warning, continue starting server (app works without context menu)
- **File not found:** Log error to console, exit 1
- **Unsupported extension:** Log error, exit 1
- **API POST failure (server running case):** Log error, exit 1
- **Server startup failure:** Log fatal error, exit 1
- **Named mutex collision at startup:** Log warning, exit 1 (prevent duplicate instances)

## Testing

- Manual testing on Windows with compiled binary
- Unit tests for registry path construction
- Unit tests for file extension validation
- Integration test: start server, POST document via API, verify record exists
