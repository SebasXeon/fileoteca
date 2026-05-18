# Folder Management — Design Spec

**Date:** 2026-05-17
**Status:** Draft

## Overview

Complete the folder ("carpetas") functionality in Fileoteca. Folders act as cross-category collections (like playlists) — a single folder can contain documents from any category/subcategory. Supports nested folder hierarchy via `parent_id`.

## Requirements

| # | Requirement |
|---|-------------|
| R1 | `/folders` page: list all folders, search/filter, create new folder (name + optional parent), delete folder |
| R2 | `/folder/[id]` page: shows folder contents (subfolders + flat document grid), breadcrumb navigation |
| R3 | Add document to folder: via right-click context menu on any `FileCard` |
| R4 | Add document to folder: via the `/document/[id]` page sidebar |
| R5 | Remove document from folder: via `/folder/[id]` page (context menu or X button) and via `/document/[id]` page sidebar |
| R6 | Documents can belong to multiple folders (many-to-many) |
| R7 | Folder hierarchy (nesting): a folder can have a parent folder |
| R8 | Document count per folder shown accurately |

## Architecture

```
┌────────────────────────────────────────────────────┐
│  SvelteKit SPA (web/src/)                          │
│                                                    │
│  /folders       /folder/[id]    ContextMenu/DocPg  │
│  Tree + Create  Docs + Subs     Folder Picker      │
│       │              │                │             │
│       ▼              ▼                ▼             │
│  ┌──────────────────────────────────────────┐      │
│  │  api/folders.ts  (PocketBase client)     │      │
│  └──────────────────────────────────────────┘      │
│       │                                            │
└───────┼────────────────────────────────────────────┘
        │  HTTP (pb.collection(...))
        ▼
┌────────────────────────────────────────────────────┐
│  PocketBase collections:                           │
│  folders, document_folders, documents              │
└────────────────────────────────────────────────────┘
```

## Data Model (already exists in DB)

### `folders` collection
| Field | Type | Notes |
|-------|------|-------|
| id | text | PK |
| name | text | Required |
| description | text | Optional |
| parent_id | relation(self) | MaxSelect: 1, nullable |
| created | autodate | |
| updated | autodate | |

### `document_folders` collection (junction)
| Field | Type | Notes |
|-------|------|-------|
| id | text | PK |
| document_id | relation(documents) | Required |
| folder_id | relation(folders) | Required |
| UNIQUE(document_id, folder_id) | | |

## API Layer Changes

### `web/src/lib/api/folders.ts`

New exports (add to existing file):

| Function | Signature | Purpose |
|----------|-----------|---------|
| `getFolder` | `(id: string) => Promise<ExplorerFolder>` | Single folder by ID |
| `getSubfolders` | `(parentId: string) => Promise<ExplorerFolder[]>` | Direct children of a folder |
| `getDocumentsInFolder` | `(folderId: string) => Promise<ExplorerFile[]>` | All documents in a folder (via junction table) |
| `addDocumentToFolder` | `(documentId: string, folderId: string) => Promise<void>` | Create junction record |
| `removeDocumentFromFolder` | `(documentId: string, folderId: string) => Promise<void>` | Delete junction record |
| `getFoldersForDocument` | `(documentId: string) => Promise<{ id: string; name: string }[]>` | Folders containing a document |
| `updateFolder` | `(id: string, data: { name?: string; description?: string; parent_id?: string \| null }) => Promise<ExplorerFolder>` | Update folder fields |

Existing functions to modify:
- `getFolders()` — query `document_folders` to compute real `count` per folder instead of hardcoded 0
- `toExplorerFolder()` — accept and pass through `description` field

### `web/src/lib/api/index.ts`
Add new exports for all new folder functions.

### `web/src/lib/types.ts`
Add `description?: string` to `ExplorerFolder` type.

## UI Pages

### `/folders` Page (`web/src/routes/folders/+page.svelte`) — Modify

```
┌─────────────────────────────────────────────┐
│  Carpetas              [Nueva Carpeta]      │
├──────────┬──────────────────────────────────┤
│ Sidebar  │                                  │
│ 🔍 Search│   Folder icon + name             │
│ [.......]│   Subfolders listed below         │
│          │   Document count badge            │
│ Folder   │   "Abrir" button -> /folder/[id] │
│ tree:    │                                  │
│  📁 A    │   When no folder selected:        │
│    📁 A1 │   "Select a folder" empty state   │
│  📁 B    │                                  │
│  📁 C    │                                  │
│   ...    │                                  │
└──────────┴──────────────────────────────────┘
```

Key behaviors:
- Left sidebar: scrollable folder tree with search filter (filter by name, keep hierarchy visible). Each folder item has a right-click context menu with "Abrir" and "Eliminar" actions, plus a delete confirmation dialog.
- Right panel: selected folder detail (name, description, subfolders, document count, "Abrir" button linking to `/folder/[id]`), or empty state
- "Nueva carpeta" button: opens `Dialog` with name input + optional parent folder dropdown (select input with `null` = root level). Parent dropdown filters out the currently selected folder to prevent self-parenting.
- Remove "Subir" button (not relevant for folders)
- Remove the grid/list toggle area (moved to the new `/folder/[id]` page)
- "Abrir" buttons actually navigate to `/folder/[id]`

### `/folder/[id]` Page (`web/src/routes/folder/[id]/+page.svelte`) — Create

```
┌─────────────────────────────────────────────┐
│  Breadcrumb: Carpetas > Folder Name          │
├─────────────────────────────────────────────┤
│  📁 Folder Name               [Abrir] [···] │
│  Folder description (if any)                 │
│                                              │
│  Subcarpetas (N)                             │
│  ┌─────────┐ ┌─────────┐                     │
│  │ 📁 Sub1 │ │ 📁 Sub2 │    (Card grid)      │
│  └─────────┘ └─────────┘                     │
│                                              │
│  Documentos (N)                              │
│  ┌──────┐ ┌──────┐ ┌──────┐                 │
│  │File  │ │File  │ │File  │   (FileCard grid)│
│  │Card  │ │Card  │ │Card  │                  │
│  └──────┘ └──────┘ └──────┘                 │
│  ┌──────┐                                    │
│  │File  │                                    │
│  │Card  │                                    │
│  └──────┘                                    │
├─────────────────────────────────────────────┤
│ "Folder" in breadcrumb                       │
└─────────────────────────────────────────────┘
```

Key behaviors:
- Load folder data: `getFolder(id)`, subfolders: `getSubfolders(id)`, documents: `getDocumentsInFolder(id)`
- Subfolders shown as clickable `Card` components (navigating to `/folder/[subId]`)
- Documents shown as flat `FileCard` grid (no category grouping)
- Empty states for both subfolders and documents ("No subcarpetas" / "No documents")
- Dropdown menu (···) on folder header: rename, delete, edit description (future/later)
- Back navigation via breadcrumb: `Carpetas > [Folder Name]`
- Document right-click context menu includes "Remove from folder" action
- "Abrir" button: navigates into folder (same behavior as clicking the card)

### Add-to-Folder Dialog (shared component)

Used in both the context menu and document page:

```
┌─────────────────────────────────┐
│  Add to Folder                  │
├─────────────────────────────────┤
│  🔍 Search folders...           │
│                                 │
│  📁 Personal                    │
│    📁 Subfolder                  │
│  📁 Work                        │
│  📁 Taxes (already added)       │
│  📁 Archive                     │
│                                 │
│  ─── or ───                     │
│  + Create new folder            │
│    Name: [________]             │
│    Parent: [optional select]    │
│                                 │
│        [Cancel]  [Add]          │
└─────────────────────────────────┘
```

Key behaviors:
- Lists all folders in a tree (indented for nesting)
- Already-added folders shown with a checkmark icon and disabled state
- "Create new folder" expands inline form (name + parent) then adds document
- Selection is single-folder (add one at a time)
- After adding, close dialog and toast success
- Search filters the folder tree by name

## Context Menu Changes

### `web/src/lib/components/explorer/document-context-menu.svelte`

Add two new context menu items:

1. **"Add to folder"** (between "Agregar a favoritos" and "Abrir externamente"):
   - Opens the shared folder picker dialog
   - Dialog is embedded in the component (self-contained, following existing pattern)

2. **"Remove from folder"** (only shown when `folderId` prop is set):
   - Calls `removeDocumentFromFolder(docId, folderId)`
   - Shown after "Add to folder" in the menu
   - Used by `/folder/[id]` page

New props: `folderId?: string` (optional, when set enables "Remove from folder" action)

New state: `showFolderDialog`. New function: loads folders, shows dialog, calls `addDocumentToFolder`.

### Document Page (`web/src/routes/document/[id]/+page.svelte`)

In the info sidebar card, add a new section after "Estado" or "Notas":

```
┌──────────────────────────────┐
│  Carpetas                    │
│  ┌─────────┐ ┌──────────┐   │
│  │ 📁 Work │ │ 📁 Pers │    │
│  │      ✕  │ │      ✕  │    │
│  └─────────┘ └──────────┘   │
│  [+ Add to folder]           │
└──────────────────────────────┘
```

- Load folders containing this document on mount via `getFoldersForDocument(docId)`
- Each folder shown as a badge with an X button to remove (calls `removeDocumentFromFolder`)
- "+ Add to folder" button opens the same folder picker dialog
- After add/remove, reload the folder badges

## Files Changed

| File | Action | Description |
|------|--------|-------------|
| `web/src/lib/api/folders.ts` | Modify | Add 7 new functions, fix `getFolders()` count, update `toExplorerFolder()` |
| `web/src/lib/api/index.ts` | Modify | Export new functions |
| `web/src/lib/types.ts` | Modify | Add `description` to `ExplorerFolder` |
| `web/src/routes/folders/+page.svelte` | Rewrite | Complete the page with working CRUD |
| `web/src/routes/folder/[id]/+page.svelte` | Create | New folder detail page |
| `web/src/lib/components/explorer/document-context-menu.svelte` | Modify | Add "Add to folder" action |
| `web/src/routes/document/[id]/+page.svelte` | Modify | Add folder badges section in sidebar |

## Error Handling & Edge Cases

- **Folder not found**: `/folder/[nonexistent]` shows "Folder not found" empty state
- **Circular parent**: PocketBase does not enforce this. The UI prevents it by filtering the current folder (and its descendants) from parent selectors in both the create folder dialog and the update folder function.
- **Double add**: `document_folders` has UNIQUE constraint on `(document_id, folder_id)` — PocketBase returns error, UI shows toast
- **Delete folder with children**: PocketBase cascade deletes `document_folders` junction records. Subfolders with `parent_id` referencing deleted folder become orphans (parent_id still points to deleted ID). Solution: on delete, also unset `parent_id` of child folders.
- **Delete folder with documents**: Junction records cascade-deleted by PocketBase. Documents remain untouched.
- **Loading states**: Each data fetch shows skeleton or spinner
- **Empty states**: "No folders yet", "No subfolders", "No documents in this folder" — each with appropriate `Empty` component and action button where applicable

## Non-Goals (Out of Scope)

- Drag-and-drop documents into folders
- Multi-select to add multiple documents at once
- Folder color/label customization
- Rename folder from context menu (can be added later)
- Folder description editing (can be added later)
- Moving documents between categories via folders
