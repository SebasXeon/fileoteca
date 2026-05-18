# Folder Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete folder ("carpetas") functionality — list, create, delete folders; view folder contents (subfolders + documents); add/remove documents from folders via context menu and document page.

**Architecture:** Extend the API layer in `folders.ts` with 7 new functions querying the `folders` and `document_folders` collections. Rewrite `/folders` page with working tree + create dialog. Create `/folder/[id]` page showing subfolders and flat document grid. Add "Add to folder" to `DocumentContextMenu` and folder badges to the document detail page.

**Tech Stack:** Svelte 5 (runes), SvelteKit (SPA), PocketBase JS client v0.26.8, shadcn-svelte, bits-ui, Tailwind CSS v4

---

### Task 1: Update Type Definitions

**Files:**
- Modify: `web/src/lib/types.ts:32-38`

- [ ] **Step 1: Add `description` field to `ExplorerFolder` type**

Read `web/src/lib/types.ts` and update the `ExplorerFolder` type:

```typescript
export type ExplorerFolder = {
	id: string;
	name: string;
	description?: string;
	parentId?: string;
	count: number;
	updatedAt: Date;
};
```

The only change is adding `description?: string;` after `name`.

- [ ] **Step 2: Verify types compile**

Run: `npx svelte-check` in `web/` directory.

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/types.ts
git commit -m "feat: add description field to ExplorerFolder type"
```

---

### Task 2: Extend Folder API Layer

**Files:**
- Modify: `web/src/lib/api/folders.ts` (complete rewrite)
- Modify: `web/src/lib/api/documents.ts` (export `toExplorerFile`)

- [ ] **Step 1: Export `toExplorerFile` from documents.ts**

Read `web/src/lib/api/documents.ts`. The function `toExplorerFile` (lines 4-28) is currently private. Add the `export` keyword:

```typescript
export function toExplorerFile(record: Record<string, unknown>): ExplorerFile {
```

Also export `mapItems` (line 30-32) since `getDocumentsInFolder` will need it:

```typescript
export function mapItems(items: Array<unknown>): ExplorerFile[] {
	return items.map((r) => toExplorerFile(r as Record<string, unknown>));
}
```

No other changes to `documents.ts`.

- [ ] **Step 2: Rewrite `web/src/lib/api/folders.ts` with all functions**

Replace the entire file content:

```typescript
import pb from "$lib/pocketbase";
import type { ExplorerFolder } from "$lib/types";
import { toExplorerFile } from "./documents";
import type { ExplorerFile } from "$lib/types";

function toExplorerFolder(record: Record<string, unknown>): ExplorerFolder {
	return {
		id: record.id as string,
		name: record.name as string,
		description: (record.description as string) ?? undefined,
		parentId: (record.parent_id as string) ?? undefined,
		count: 0,
		updatedAt: new Date(record.updated as string),
	};
}

export async function getFolders(): Promise<ExplorerFolder[]> {
	const [folderRecords, junctionRecords] = await Promise.all([
		pb.collection("folders").getFullList({ sort: "name" }),
		pb.collection("document_folders").getFullList(),
	]);

	const countMap: Record<string, number> = {};
	for (const jr of junctionRecords) {
		const fid = (jr as unknown as Record<string, unknown>).folder_id as string;
		countMap[fid] = (countMap[fid] || 0) + 1;
	}

	return folderRecords.map((r) => {
		const raw = r as unknown as Record<string, unknown>;
		return {
			id: raw.id as string,
			name: raw.name as string,
			description: (raw.description as string) ?? undefined,
			parentId: (raw.parent_id as string) ?? undefined,
			count: countMap[raw.id as string] || 0,
			updatedAt: new Date(raw.updated as string),
		};
	});
}

export async function getFolder(id: string): Promise<ExplorerFolder> {
	const [record, junctionRecords] = await Promise.all([
		pb.collection("folders").getOne(id),
		pb.collection("document_folders").getFullList({ filter: `folder_id = "${id}"` }),
	]);
	const raw = record as unknown as Record<string, unknown>;
	return {
		id: raw.id as string,
		name: raw.name as string,
		description: (raw.description as string) ?? undefined,
		parentId: (raw.parent_id as string) ?? undefined,
		count: junctionRecords.length,
		updatedAt: new Date(raw.updated as string),
	};
}

export async function getSubfolders(parentId: string): Promise<ExplorerFolder[]> {
	const records = await pb.collection("folders").getFullList({
		filter: `parent_id = "${parentId}"`,
		sort: "name",
	});
	return records.map((r) => toExplorerFolder(r as unknown as Record<string, unknown>));
}

export async function getDocumentsInFolder(folderId: string): Promise<ExplorerFile[]> {
	const junctionRecords = await pb.collection("document_folders").getFullList({
		filter: `folder_id = "${folderId}"`,
	});
	const docIds = junctionRecords
		.map((r) => (r as unknown as Record<string, unknown>).document_id as string)
		.filter(Boolean);

	if (docIds.length === 0) return [];

	const filter = docIds.map((id) => `id = "${id}"`).join(" || ");
	const result = await pb.collection("documents").getList(1, docIds.length, {
		filter,
		sort: "-created",
		expand: "category_id,subcategory_id",
	});
	return (result.items as Array<unknown>).map((r) =>
		toExplorerFile(r as Record<string, unknown>),
	);
}

export async function createFolder(name: string, parentId?: string): Promise<ExplorerFolder> {
	const data: Record<string, unknown> = { name };
	if (parentId) data.parent_id = parentId;
	const record = await pb.collection("folders").create(data);
	return toExplorerFolder(record as unknown as Record<string, unknown>);
}

export async function updateFolder(
	id: string,
	data: { name?: string; description?: string; parent_id?: string | null },
): Promise<ExplorerFolder> {
	const record = await pb.collection("folders").update(id, data);
	return toExplorerFolder(record as unknown as Record<string, unknown>);
}

export async function deleteFolder(id: string): Promise<void> {
	// Unset parent_id on children so they don't become orphans
	const children = await pb.collection("folders").getFullList({
		filter: `parent_id = "${id}"`,
	});
	for (const child of children) {
		await pb.collection("folders").update(child.id, { parent_id: null });
	}
	await pb.collection("folders").delete(id);
}

export async function addDocumentToFolder(documentId: string, folderId: string): Promise<void> {
	await pb.collection("document_folders").create({
		document_id: documentId,
		folder_id: folderId,
	});
}

export async function removeDocumentFromFolder(documentId: string, folderId: string): Promise<void> {
	const records = await pb.collection("document_folders").getFullList({
		filter: `document_id = "${documentId}" && folder_id = "${folderId}"`,
	});
	if (records.length > 0) {
		await pb.collection("document_folders").delete(records[0].id);
	}
}

export async function getFoldersForDocument(
	documentId: string,
): Promise<{ id: string; name: string }[]> {
	const records = await pb.collection("document_folders").getFullList({
		filter: `document_id = "${documentId}"`,
		expand: "folder_id",
	});
	return records.map((r) => {
		const raw = r as unknown as Record<string, unknown>;
		const expand = raw.expand as Record<string, Record<string, unknown>> | undefined;
		const folder = expand?.folder_id;
		return {
			id: raw.folder_id as string,
			name: (folder?.name as string) || "Unknown",
		};
	});
}
```

- [ ] **Step 3: Run svelte-check**

Run: `npx svelte-check` in `web/` directory.

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/api/folders.ts web/src/lib/api/documents.ts
git commit -m "feat: extend folder API with document-folder operations and count fix"
```

---

### Task 3: Update API Exports

**Files:**
- Modify: `web/src/lib/api/index.ts`

- [ ] **Step 1: Add new exports to index.ts**

Read `web/src/lib/api/index.ts` and update line 4 to export all new functions:

```typescript
export { default as pb } from "$lib/pocketbase";
export { getRecentDocuments, getFavoriteDocuments, getSuggestedDocuments, searchDocuments, deleteDocument, getDocument, openDocument, getOpenUrl, toggleFavorite, updateDocumentCategory, updateDocumentNotes, getCategoriesWithSubcategories, type DocumentDetail } from "./documents";
export { getCategories, getIcons, createCategory, getDocumentsByCategory } from "./categories";
export { getFolders, getFolder, getSubfolders, getDocumentsInFolder, createFolder, updateFolder, deleteFolder, addDocumentToFolder, removeDocumentFromFolder, getFoldersForDocument } from "./folders";
```

- [ ] **Step 2: Run svelte-check**

Run: `npx svelte-check` in `web/` directory.

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/api/index.ts
git commit -m "feat: export new folder API functions"
```

---

### Task 4: Add Folder Actions to Document Context Menu

**Files:**
- Modify: `web/src/lib/components/explorer/document-context-menu.svelte`

- [ ] **Step 1: Add folder-related props, state, imports, and dialog to the context menu**

Read `web/src/lib/components/explorer/document-context-menu.svelte`. Add the new `folderId` prop, new imports, new state, new handler functions, dialog, and menu items.

Add to the `<script>` block imports (after existing imports):

```typescript
import FolderPlusIcon from "@lucide/svelte/icons/folder-plus";
import FolderMinusIcon from "@lucide/svelte/icons/folder-minus";
import FolderIcon from "@lucide/svelte/icons/folder";
import CheckIcon from "@lucide/svelte/icons/check";
import PlusIcon from "@lucide/svelte/icons/plus";
import SearchIcon from "@lucide/svelte/icons/search";
import * as Dialog from "$lib/components/ui/dialog/index.js";
import { Input } from "$lib/components/ui/input/index.js";
import { Label } from "$lib/components/ui/label/index.js";
import { Separator } from "$lib/components/ui/separator/index.js";
import {
	addDocumentToFolder,
	removeDocumentFromFolder,
	getFolders,
	createFolder,
} from "$lib/api";
import type { ExplorerFolder } from "$lib/types";
```

Add new prop to the destructured props:

```typescript
let {
	docId,
	docName = "",
	isFavorite = false,
	isVisualizable = false,
	folderId,
	onchange,
	onFavoriteToggle,
	children,
}: {
	docId: string;
	docName?: string;
	isFavorite?: boolean;
	isVisualizable?: boolean;
	folderId?: string;
	onchange: () => void;
	onFavoriteToggle?: (isFavorite: boolean) => void;
	children: Snippet;
} = $props();
```

Add new state variables after the existing `let categories` line:

```typescript
let showFolderDialog = $state(false);
let folderDialogQuery = $state("");
let folderList = $state<ExplorerFolder[]>([]);
let selectedFolderId = $state<string | "">("");
let creatingFolder = $state(false);
let newFolderName = $state("");
let newFolderParentId = $state<string | "">("");
```

Add new handler functions (after `handleCopyPath`, before `</script>`):

```typescript
async function loadFolders() {
	try {
		const folders = await getFolders();
		folderList = folders;
	} catch (err) {
		console.error("Failed to load folders:", err);
	}
}

function openFolderDialog() {
	folderDialogQuery = "";
	selectedFolderId = "";
	creatingFolder = false;
	newFolderName = "";
	newFolderParentId = "";
	showFolderDialog = true;
	loadFolders();
}

const visibleFolders = $derived.by(() => {
	const q = folderDialogQuery.trim().toLowerCase();
	if (!q) return folderList;
	return folderList.filter((f) => f.name.toLowerCase().includes(q));
});

// Build a flat, indented list from the folder tree
function flattenFolders(folders: ExplorerFolder[], parentId: string | null = null, level = 0): { folder: ExplorerFolder; level: number }[] {
	const result: { folder: ExplorerFolder; level: number }[] = [];
	const items = folders.filter((f) => (parentId === null ? !f.parentId : f.parentId === parentId));
	for (const f of items) {
		result.push({ folder: f, level });
		result.push(...flattenFolders(folders, f.id, level + 1));
	}
	return result;
}

// Check if document is already in a folder
let docFolderIds = $state<string[]>([]);

async function loadDocFolderIds() {
	try {
		const folders = await (await import("$lib/api")).getFoldersForDocument(docId);
		docFolderIds = folders.map((f) => f.id);
	} catch {
		docFolderIds = [];
	}
}

// Load doc folders when dialog opens
$effect(() => {
	if (showFolderDialog) {
		loadDocFolderIds();
	}
});

async function handleAddToFolder() {
	if (!selectedFolderId) return;
	try {
		await addDocumentToFolder(docId, selectedFolderId);
		toast.success("Documento agregado a la carpeta");
		showFolderDialog = false;
		onchange();
	} catch (err: any) {
		if (err.message?.includes("duplicate") || err.message?.includes("unique")) {
			toast.error("El documento ya está en esta carpeta");
		} else {
			toast.error(`Error: ${err}`);
		}
	}
}

async function handleRemoveFromFolder() {
	if (!folderId) return;
	try {
		await removeDocumentFromFolder(docId, folderId);
		toast.success("Documento removido de la carpeta");
		onchange();
	} catch (err) {
		toast.error(`Error: ${err}`);
	}
}

async function handleCreateAndAdd() {
	if (!newFolderName.trim()) return;
	try {
		const folder = await createFolder(
			newFolderName.trim(),
			newFolderParentId || undefined,
		);
		await addDocumentToFolder(docId, folder.id);
		toast.success("Carpeta creada y documento agregado");
		showFolderDialog = false;
		onchange();
	} catch (err) {
		toast.error(`Error: ${err}`);
	}
}
```

Add the new menu items to the `ContextMenu.Content` section. Insert after the favorite toggle item (`</ContextMenu.Item>` after `handleToggleFavorite`):

```svelte
<ContextMenu.Item onclick={openFolderDialog}>
	<FolderPlusIcon class="mr-2 size-4" />
	Agregar a carpeta
</ContextMenu.Item>
{#if folderId}
	<ContextMenu.Item onclick={handleRemoveFromFolder}>
		<FolderMinusIcon class="mr-2 size-4" />
		Remover de carpeta
	</ContextMenu.Item>
{/if}
```

Add the folder picker dialog at the end of the component (after `</ContextMenu.Root>`, inside the file):

```svelte
<Dialog.Root open={showFolderDialog} onOpenChange={(o) => { showFolderDialog = o; if (!o) { creatingFolder = false; } }}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Agregar a carpeta</Dialog.Title>
			<Dialog.Description>
				Selecciona una carpeta para "{docName}"
			</Dialog.Description>
		</Dialog.Header>
		<div class="flex flex-col gap-4 py-4">
			<div class="relative">
				<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
				<Input
					bind:value={folderDialogQuery}
					placeholder="Buscar carpetas..."
					class="pl-9"
				/>
			</div>

			<div class="max-h-60 overflow-y-auto flex flex-col gap-1">
				{#each flattenFolders(visibleFolders) as { folder, level } (folder.id)}
					{@const isAlreadyIn = docFolderIds.includes(folder.id)}
					<button
						type="button"
						class="hover:bg-muted/50 data-[active=true]:bg-muted/80 flex w-full items-center justify-between gap-3 rounded-3xl px-3 py-2 text-left text-sm"
						data-active={selectedFolderId === folder.id}
						style="padding-left: {12 + level * 16}px"
						disabled={isAlreadyIn}
						onclick={() => { selectedFolderId = folder.id; }}
					>
						<div class="flex min-w-0 items-center gap-2">
							<FolderIcon class="text-muted-foreground size-4 shrink-0" />
							<span class="truncate">{folder.name}</span>
						</div>
						{#if isAlreadyIn}
							<CheckIcon class="text-muted-foreground size-4 shrink-0" />
						{/if}
					</button>
				{/each}
				{#if visibleFolders.length === 0}
					<p class="text-muted-foreground py-4 text-center text-sm">No se encontraron carpetas</p>
				{/if}
			</div>

			<Separator />

			{#if !creatingFolder}
				<Button variant="ghost" size="sm" class="justify-start" onclick={() => { creatingFolder = true; }}>
					<PlusIcon class="mr-2 size-4" />
					Crear nueva carpeta
				</Button>
			{:else}
				<div class="flex flex-col gap-3 rounded-2xl border p-3">
					<div class="flex flex-col gap-2">
						<Label for="new-folder-name">Nombre</Label>
						<Input id="new-folder-name" bind:value={newFolderName} placeholder="Nombre de la carpeta" />
					</div>
					<div class="flex flex-col gap-2">
						<Label for="new-folder-parent">Carpeta padre (opcional)</Label>
						<select
							id="new-folder-parent"
							bind:value={newFolderParentId}
							class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-10 w-full rounded-2xl border px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
						>
							<option value="">Ninguna (raíz)</option>
							{#each folderList as f (f.id)}
								<option value={f.id}>{f.name}</option>
							{/each}
						</select>
					</div>
					<div class="flex gap-2 justify-end">
						<Button variant="outline" size="sm" onclick={() => { creatingFolder = false; }}>
							Cancelar
						</Button>
						<Button size="sm" onclick={handleCreateAndAdd} disabled={!newFolderName.trim()}>
							Crear y agregar
						</Button>
					</div>
				</div>
			{/if}
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => { showFolderDialog = false; }}>
				Cancelar
			</Button>
			<Button onclick={handleAddToFolder} disabled={!selectedFolderId}>
				Agregar
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
```

- [ ] **Step 2: Run svelte-check**

Run: `npx svelte-check` in `web/` directory.

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/components/explorer/document-context-menu.svelte
git commit -m "feat: add 'Add to folder' and 'Remove from folder' to document context menu"
```

---

### Task 5: Rewrite `/folders` Page

**Files:**
- Modify: `web/src/routes/folders/+page.svelte` (complete rewrite)

- [ ] **Step 1: Replace the entire file**

Replace `web/src/routes/folders/+page.svelte` with:

```svelte
<script lang="ts">
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import SearchIcon from "@lucide/svelte/icons/search";
	import FolderPlusIcon from "@lucide/svelte/icons/folder-plus";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import MoreHorizontalIcon from "@lucide/svelte/icons/ellipsis";
	import { toast } from "svelte-sonner";

	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import { Empty } from "$lib/components/ui/empty/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Label } from "$lib/components/ui/label/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";

	import { FolderIcon, formatDate, type ExplorerFolder } from "$lib/types";
	import { getFolders, createFolder, deleteFolder } from "$lib/api";

	let folders = $state<ExplorerFolder[]>([]);
	let loading = $state(true);
	let query = $state("");
	let selectedFolderId = $state<string | null>(null);
	let showCreateDialog = $state(false);
	let newName = $state("");
	let newParentId = $state<string | "">("");
	let folderToDelete = $state<ExplorerFolder | null>(null);

	onMount(async () => {
		await loadFolders();
	});

	async function loadFolders() {
		try {
			folders = await getFolders();
		} catch (err) {
			console.error("Failed to load folders:", err);
		} finally {
			loading = false;
		}
	}

	const visibleFolders = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return folders;
		return folders.filter((f) => f.name.toLowerCase().includes(q));
	});

	const selectedFolder: ExplorerFolder | null = $derived.by(() => {
		if (!selectedFolderId) return null;
		return folders.find((f) => f.id === selectedFolderId) ?? null;
	});

	const selectedSubfolders = $derived.by(() => {
		if (!selectedFolderId) return [];
		return folders.filter((f) => f.parentId === selectedFolderId);
	});

	// For the tree sidebar: show root-level folders only
	const rootFolders = $derived.by(() => {
		return visibleFolders.filter((f) => !f.parentId);
	});

	function getChildFolders(parentId: string): ExplorerFolder[] {
		return visibleFolders.filter((f) => f.parentId === parentId);
	}

	// Build hierarchical tree for sidebar display (recursive)
	const folderTree = $derived.by(() => {
		function buildTree(parentId: string | null): { folder: ExplorerFolder; children: { folder: ExplorerFolder; children: any[] }[] }[] {
			const items = visibleFolders.filter((f) => (parentId === null ? !f.parentId : f.parentId === parentId));
			return items.map((f) => ({
				folder: f,
				children: buildTree(f.id),
			}));
		}
		return buildTree(null);
	});

	// Check if a folder is a descendant of another (for preventing circular parents)
	function isDescendant(folderId: string, potentialAncestorId: string): boolean {
		const children = folders.filter((f) => f.parentId === potentialAncestorId);
		for (const child of children) {
			if (child.id === folderId) return true;
			if (isDescendant(folderId, child.id)) return true;
		}
		return false;
	}

	const availableParents = $derived.by(() => {
		if (!selectedFolderId) return folders;
		return folders.filter(
			(f) => f.id !== selectedFolderId && !isDescendant(f.id, selectedFolderId),
		);
	});

	async function handleCreateFolder() {
		const name = newName.trim();
		if (!name) return;
		try {
			await createFolder(name, newParentId || undefined);
			toast.success("Carpeta creada");
			showCreateDialog = false;
			newName = "";
			newParentId = "";
			await loadFolders();
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleDeleteFolder() {
		if (!folderToDelete) return;
		try {
			await deleteFolder(folderToDelete.id);
			if (selectedFolderId === folderToDelete.id) {
				selectedFolderId = null;
			}
			toast.success("Carpeta eliminada");
			folderToDelete = null;
			await loadFolders();
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	function openCreateDialog() {
		newName = "";
		newParentId = "";
		showCreateDialog = true;
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="min-w-0">
			<h1 class="text-2xl font-semibold leading-tight">Carpetas</h1>
			<p class="text-muted-foreground">
				Agrupa documentos de cualquier categoría en carpetas.
			</p>
		</div>
		<div class="flex items-center gap-2">
			<Button onclick={openCreateDialog}>
				<FolderPlusIcon class="mr-2 size-4" />
				Nueva carpeta
			</Button>
		</div>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-10 text-center">Cargando…</div>
	{:else if folders.length === 0}
		<Empty>
			<div class="text-muted-foreground">
				Aún no tienes carpetas. Crea una para agrupar documentos.
			</div>
			<Button onclick={openCreateDialog}>
				<FolderPlusIcon class="mr-2 size-4" />
				Crear carpeta
			</Button>
		</Empty>
	{:else}
		<div class="grid gap-4 lg:grid-cols-[320px_1fr]">
			<Card.Root class="shadow-sm">
				<Card.Header class="pb-2">
					<Card.Title class="flex items-center gap-2">
						<FolderIcon class="text-muted-foreground size-4" />
						Carpetas
					</Card.Title>
					<Card.Description>Selecciona para ver contenido.</Card.Description>
				</Card.Header>
				<Card.Content class="pt-0">
					<div class="flex flex-col gap-3">
						<div class="relative">
							<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
							<Input bind:value={query} placeholder="Buscar carpetas…" class="pl-9" />
						</div>

						<div class="flex items-center justify-between text-sm">
							<span class="text-muted-foreground">{visibleFolders.length} carpeta(s)</span>
						</div>

						<Separator />

						<div class="flex flex-col gap-1 max-h-[60vh] overflow-y-auto">
							{#each folderTree as { folder, children } (folder.id)}
								<div>
									<button
										type="button"
										class="hover:bg-muted/50 data-[active=true]:bg-muted/80 flex w-full items-center justify-between gap-3 rounded-3xl px-3 py-2 text-left text-sm"
										data-active={selectedFolderId === folder.id}
										onclick={() => { selectedFolderId = folder.id; }}
									>
										<div class="flex min-w-0 items-center gap-2">
											<FolderIcon class="text-muted-foreground size-4 shrink-0" />
											<span class="truncate font-medium">{folder.name}</span>
										</div>
										<div class="flex items-center gap-1 shrink-0">
											<Badge variant="secondary">{folder.count}</Badge>
											<DropdownMenu.Root>
												<DropdownMenu.Trigger>
													{#snippet child({ props })}
														<button
															{...props}
															aria-label="Más opciones"
															class="hover:bg-muted size-6 flex items-center justify-center rounded-full"
															onclick={(e) => e.stopPropagation()}
														>
															<MoreHorizontalIcon class="size-3" />
														</button>
													{/snippet}
												</DropdownMenu.Trigger>
												<DropdownMenu.Content align="end" class="min-w-40">
													<DropdownMenu.Item onclick={() => goto(`/folder/${folder.id}`)}>
														Abrir
													</DropdownMenu.Item>
													<DropdownMenu.Separator />
													<DropdownMenu.Item
														class="text-destructive"
														onclick={() => { folderToDelete = folder; }}
													>
														<TrashIcon class="mr-2 size-4" />
														Eliminar
													</DropdownMenu.Item>
												</DropdownMenu.Content>
											</DropdownMenu.Root>
										</div>
									</button>
									{#if children.length > 0}
										<div class="pl-5">
											{#each children as { folder: childFolder, children: grandChildren } (childFolder.id)}
												<button
													type="button"
													class="hover:bg-muted/50 data-[active=true]:bg-muted/80 flex w-full items-center justify-between gap-3 rounded-3xl px-3 py-1.5 text-left text-sm"
													data-active={selectedFolderId === childFolder.id}
													onclick={() => { selectedFolderId = childFolder.id; }}
												>
													<div class="flex min-w-0 items-center gap-2">
														<FolderIcon class="text-muted-foreground size-3.5 shrink-0" />
														<span class="truncate">{childFolder.name}</span>
													</div>
													<Badge variant="secondary">{childFolder.count}</Badge>
												</button>
											{/each}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					</div>
				</Card.Content>
			</Card.Root>

			<div class="flex flex-col gap-4">
				{#if !selectedFolder}
					<Empty>
						<div class="text-muted-foreground">Selecciona una carpeta para ver su contenido.</div>
					</Empty>
				{:else}
					<Card.Root class="shadow-sm">
						<Card.Header>
							<div class="flex items-center justify-between">
								<div>
									<Card.Title class="flex items-center gap-2">
										<FolderIcon class="text-muted-foreground size-5" />
										{selectedFolder.name}
									</Card.Title>
									<Card.Description>
										{#if selectedFolder.description}{selectedFolder.description}{:else}Sin descripción{/if}
									</Card.Description>
								</div>
								<Button variant="outline" size="sm" onclick={() => goto(`/folder/${selectedFolder.id}`)}>
									Abrir
								</Button>
							</div>
						</Card.Header>
						<Card.Content>
							<div class="flex items-center gap-4 text-sm">
								<div class="flex items-center gap-1">
									<FolderIcon class="text-muted-foreground size-4" />
									<span>{selectedSubfolders.length} subcarpeta(s)</span>
								</div>
								<Separator orientation="vertical" class="h-4" />
								<div>
									<span>{selectedFolder.count} documento(s)</span>
								</div>
							</div>
						</Card.Content>
					</Card.Root>

					{#if selectedSubfolders.length > 0}
						<div>
							<h3 class="text-sm font-medium mb-2 text-muted-foreground">Subcarpetas</h3>
							<div class="grid gap-3 sm:grid-cols-2">
								{#each selectedSubfolders as sub (sub.id)}
									<Card.Root class="shadow-sm">
										<Card.Header class="pb-2">
											<div class="flex items-start justify-between gap-3">
												<div class="min-w-0">
													<Card.Title class="flex items-center gap-2">
														<FolderIcon class="text-muted-foreground size-4" />
														<span class="truncate">{sub.name}</span>
													</Card.Title>
													<Card.Description class="truncate">
														{sub.count} documento(s) · {formatDate(sub.updatedAt)}
													</Card.Description>
												</div>
												<Button variant="ghost" size="sm" onclick={() => goto(`/folder/${sub.id}`)}>Abrir</Button>
											</div>
										</Card.Header>
									</Card.Root>
								{/each}
							</div>
						</div>
					{/if}

					{#if selectedFolder.count > 0}
						<div class="text-muted-foreground text-sm">
							{selectedFolder.count} documento(s) en esta carpeta. <a href="/folder/{selectedFolder.id}" class="underline hover:text-foreground">Ver todos</a>
						</div>
					{/if}
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Create Folder Dialog -->
<Dialog.Root open={showCreateDialog} onOpenChange={(o) => { showCreateDialog = o; }}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Nueva carpeta</Dialog.Title>
			<Dialog.Description>
				Crea una carpeta para agrupar documentos.
			</Dialog.Description>
		</Dialog.Header>
		<div class="flex flex-col gap-4 py-4">
			<div class="flex flex-col gap-2">
				<Label for="folder-name">Nombre</Label>
				<Input id="folder-name" bind:value={newName} placeholder="Nombre de la carpeta" />
			</div>
			<div class="flex flex-col gap-2">
				<Label for="folder-parent">Carpeta padre (opcional)</Label>
				<select
					id="folder-parent"
					bind:value={newParentId}
					class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-10 w-full rounded-2xl border px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
				>
					<option value="">Ninguna (raíz)</option>
					{#each folders as f (f.id)}
						<option value={f.id}>{f.name}</option>
					{/each}
				</select>
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => { showCreateDialog = false; }}>
				Cancelar
			</Button>
			<Button onclick={handleCreateFolder} disabled={!newName.trim()}>
				Crear
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Delete Confirmation Dialog -->
<Dialog.Root open={!!folderToDelete} onOpenChange={(o) => { if (!o) folderToDelete = null; }}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Eliminar carpeta</Dialog.Title>
			<Dialog.Description>
				¿Eliminar "{folderToDelete?.name}"? Las subcarpetas quedarán en la raíz. Los documentos no se eliminarán.
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => { folderToDelete = null; }}>
				Cancelar
			</Button>
			<Button variant="destructive" onclick={handleDeleteFolder}>
				Eliminar
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
```

- [ ] **Step 2: Run svelte-check**

Run: `npx svelte-check` in `web/` directory.

- [ ] **Step 3: Commit**

```bash
git add web/src/routes/folders/+page.svelte
git commit -m "feat: rewrite /folders page with create, delete, tree navigation"
```

---

### Task 6: Create `/folder/[id]` Page

**Files:**
- Create: `web/src/routes/folder/[id]/+page.svelte`

- [ ] **Step 1: Create the directory and file**

Check parent exists: `ls web/src/routes/folder`. Create the directory and file:

```bash
mkdir -p web/src/routes/folder/\[id\]
```

- [ ] **Step 2: Write the page**

Write `web/src/routes/folder/[id]/+page.svelte`:

```svelte
<script lang="ts">
	import { onMount } from "svelte";
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import FolderIcon from "@lucide/svelte/icons/folder";
	import LayoutGridIcon from "@lucide/svelte/icons/layout-grid";
	import ListIcon from "@lucide/svelte/icons/list";
	import SearchIcon from "@lucide/svelte/icons/search";
	import XIcon from "@lucide/svelte/icons/x";

	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { ButtonGroup } from "$lib/components/ui/button-group/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Item } from "$lib/components/ui/item/index.js";
	import { Empty } from "$lib/components/ui/empty/index.js";

	import FileCard from "$lib/components/explorer/file-card.svelte";
	import DocumentContextMenu from "$lib/components/explorer/document-context-menu.svelte";

	import { fileIconFor, formatBytes, formatDate, type ExplorerFolder, type ExplorerFile } from "$lib/types";
	import { getFolder, getSubfolders, getDocumentsInFolder } from "$lib/api";

	let folderId = $state("");
	let folderName = $state("");
	let folderDesc = $state("");
	let folderCount = $state(0);
	let subfolders = $state<ExplorerFolder[]>([]);
	let documents = $state<ExplorerFile[]>([]);
	let loading = $state(true);
	let query = $state("");
	let debouncedQuery = $state("");
	let view: "grid" | "list" = $state("grid");

	$effect(() => {
		const val = query;
		const timer = setTimeout(() => { debouncedQuery = val; }, 250);
		return () => clearTimeout(timer);
	});

	onMount(async () => {
		folderId = $page.params.id ?? "";
		if (!folderId) return;
		await loadData();
	});

	async function loadData() {
		loading = true;
		try {
			const [folder, subs, docs] = await Promise.all([
				getFolder(folderId),
				getSubfolders(folderId),
				getDocumentsInFolder(folderId),
			]);
			folderName = folder.name;
			folderDesc = folder.description ?? "";
			folderCount = folder.count;
			subfolders = subs;
			documents = docs;
		} catch (err) {
			console.error("Failed to load folder:", err);
			toast.error("No se pudo cargar la carpeta");
		} finally {
			loading = false;
		}
	}

	const filteredDocs = $derived.by(() => {
		const q = debouncedQuery.trim().toLowerCase();
		if (!q) return documents;
		return documents.filter((d) =>
			d.name.toLowerCase().includes(q) ||
			(d.locationLabel ?? "").toLowerCase().includes(q)
		);
	});

	const visualizable = (ext: string) =>
		["pdf", "png", "jpg", "jpeg", "gif", "bmp", "svg", "webp", "tiff", "ico", "txt", "csv", "md", "json", "xml", "html", "htm", "rtf"].includes(ext);

	async function handleDocumentChange() {
		await loadData();
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center gap-2">
		<Button variant="ghost" size="icon-sm" onclick={() => goto("/folders")}>
			<ArrowLeftIcon class="size-4" />
		</Button>
		<div class="flex items-center gap-3">
			<FolderIcon class="text-muted-foreground size-5" />
			<h1 class="text-2xl font-semibold leading-tight">{folderName}</h1>
		</div>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-10 text-center">Cargando…</div>
	{:else}
		<div class="flex flex-col gap-6">
			{#if folderDesc}
				<p class="text-muted-foreground text-sm">{folderDesc}</p>
			{/if}

			{#if subfolders.length > 0}
				<div class="flex flex-col gap-3">
					<h2 class="text-sm font-medium text-muted-foreground">Subcarpetas ({subfolders.length})</h2>
					<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
						{#each subfolders as sub (sub.id)}
							<Card.Root class="shadow-sm cursor-pointer hover:bg-muted/30 transition-colors" onclick={() => goto(`/folder/${sub.id}`)}>
								<Card.Header class="pb-2">
									<div class="min-w-0">
										<Card.Title class="flex items-center gap-2">
											<FolderIcon class="text-muted-foreground size-4" />
											<span class="truncate">{sub.name}</span>
										</Card.Title>
										<Card.Description class="truncate">
											{sub.count} documento(s) · {formatDate(sub.updatedAt)}
										</Card.Description>
									</div>
								</Card.Header>
							</Card.Root>
						{/each}
					</div>
				</div>
			{/if}

			<div class="flex flex-col gap-4">
				<div class="flex flex-wrap items-center gap-2">
					<div class="relative min-w-[260px] flex-1">
						<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
						<Input bind:value={query} placeholder="Buscar documentos en esta carpeta…" class="pl-9" />
						{#if query}
							<button
								class="absolute right-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground hover:text-foreground"
								onclick={() => { query = ""; }}
							>
								<XIcon class="size-4" />
							</button>
						{/if}
					</div>
					<ButtonGroup>
						<Button variant="outline" size="icon-sm" aria-label="Grid" data-active={view === "grid"} onclick={() => { view = "grid"; }}>
							<LayoutGridIcon class="size-4" />
						</Button>
						<Button variant="outline" size="icon-sm" aria-label="Lista" data-active={view === "list"} onclick={() => { view = "list"; }}>
							<ListIcon class="size-4" />
						</Button>
					</ButtonGroup>
				</div>

				<div class="flex items-center gap-2">
					<Badge variant="secondary">Documentos</Badge>
					<span class="text-muted-foreground text-sm">{filteredDocs.length} documento(s)</span>
				</div>

				{#if filteredDocs.length === 0}
					<Empty>
						<div class="text-muted-foreground">
							{#if query}
								Sin resultados con los filtros actuales.
							{:else}
								No hay documentos en esta carpeta. Agrega documentos usando el menú contextual (clic derecho en cualquier documento).
							{/if}
						</div>
					</Empty>
				{:else if view === "grid"}
					<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
						{#each filteredDocs as file (file.id)}
							<FileCard {file} href={`/document/${file.id}`} onchange={handleDocumentChange} />
						{/each}
					</div>
				{:else}
					<div class="flex flex-col">
						{#each filteredDocs as file (file.id)}
							{@const Icon = fileIconFor(file.ext)}
							<DocumentContextMenu
								docId={file.id}
								docName={file.name}
								isFavorite={file.favorite ?? false}
								isVisualizable={visualizable(file.ext)}
								folderId={folderId}
								onchange={handleDocumentChange}
							>
								{#snippet children()}
									<Item variant="muted" size="sm" class="rounded-3xl cursor-pointer hover:bg-muted/80" onclick={() => goto(`/document/${file.id}`)}>
										<div class="flex w-full items-center gap-3">
											<div class="bg-muted flex size-9 items-center justify-center rounded-2xl overflow-hidden">
												{#if file.thumbnail}
													<img src={file.thumbnail} alt="" class="size-full object-cover" />
												{:else}
													<Icon class="text-muted-foreground size-4" />
												{/if}
											</div>
											<div class="min-w-0 flex-1">
												<div class="truncate font-medium">{file.name}</div>
												<div class="text-muted-foreground truncate text-xs">
													{file.locationLabel}
												</div>
											</div>
											<div class="text-muted-foreground hidden text-xs md:block">
												{formatDate(file.updatedAt)}
											</div>
											<div class="text-muted-foreground w-[90px] text-end text-xs">
												{formatBytes(file.sizeBytes)}
											</div>
										</div>
									</Item>
								{/snippet}
							</DocumentContextMenu>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
```

- [ ] **Step 3: Run svelte-check**

Run: `npx svelte-check` in `web/` directory.

- [ ] **Step 4: Commit**

```bash
git add web/src/routes/folder/
git commit -m "feat: create /folder/[id] page with subfolders and document grid"
```

---

### Task 7: Add Folder Badges to Document Page

**Files:**
- Modify: `web/src/routes/document/[id]/+page.svelte`

- [ ] **Step 1: Add folder badges section to the document page sidebar**

Read the existing file at `web/src/routes/document/[id]/+page.svelte`. Add the following new imports to the existing import blocks:

**Add to lucide icons imports** (after line 13, before `PlusIcon` is not there yet — add these new lucide icons):
```typescript
import XIcon from "@lucide/svelte/icons/x";
import PlusIcon from "@lucide/svelte/icons/plus";
import SearchIcon from "@lucide/svelte/icons/search";
import CheckIcon from "@lucide/svelte/icons/check";
```

**Add to $lib/components/ui imports** (after existing UI imports around line 16-19):
```typescript
import * as Dialog from "$lib/components/ui/dialog/index.js";
import { Input } from "$lib/components/ui/input/index.js";
import { Label } from "$lib/components/ui/label/index.js";
```

**Update the `$lib/api` import** (line 20-28): add the new functions to the existing import block:
```typescript
getFoldersForDocument, removeDocumentFromFolder, addDocumentToFolder, getFolders, createFolder,
```

**Update the `$lib/types` import** (line 30): add `FolderIcon` and `ExplorerFolder`:
```typescript
import { FolderIcon, formatBytes, formatDate, type ExplorerFolder } from "$lib/types";
```

Add new state variables (after the existing `let textContent` line):

```typescript
let docFolders = $state<{ id: string; name: string }[]>([]);
let showFolderDialog = $state(false);
let folderDialogQuery = $state("");
let folderList = $state<ExplorerFolder[]>([]);
let selectedFolderId = $state<string | "">("");
let creatingFolder = $state(false);
let newFolderName = $state("");
let newFolderParentId = $state<string | "">("");
```

Update `onMount` to also load document folders:

```typescript
onMount(async () => {
	await loadDocument();
	getCategoriesWithSubcategories().then((cats) => { categories = cats; }).catch(() => {});
	loadDocumentFolders();
});
```

Add folder-related helper functions (after existing handlers, before `</script>`):

```typescript
async function loadDocumentFolders() {
	if (!id) return;
	try {
		docFolders = await getFoldersForDocument(id);
	} catch (err) {
		console.error("Failed to load document folders:", err);
	}
}

async function handleRemoveFolder(folderId: string) {
	if (!id) return;
	try {
		await removeDocumentFromFolder(id, folderId);
		docFolders = docFolders.filter((f) => f.id !== folderId);
		toast.success("Documento removido de la carpeta");
	} catch (err) {
		toast.error(`Error: ${err}`);
	}
}

async function loadFolderList() {
	try {
		folderList = await getFolders();
	} catch (err) {
		console.error("Failed to load folders:", err);
	}
}

function openFolderDialog() {
	folderDialogQuery = "";
	selectedFolderId = "";
	creatingFolder = false;
	newFolderName = "";
	newFolderParentId = "";
	showFolderDialog = true;
	loadFolderList();
}

const visibleDialogFolders = $derived.by(() => {
	const q = folderDialogQuery.trim().toLowerCase();
	if (!q) return folderList;
	return folderList.filter((f) => f.name.toLowerCase().includes(q));
});

function flattenFolders(folders: ExplorerFolder[], parentId: string | null = null, level = 0): { folder: ExplorerFolder; level: number }[] {
	const result: { folder: ExplorerFolder; level: number }[] = [];
	const items = folders.filter((f) => (parentId === null ? !f.parentId : f.parentId === parentId));
	for (const f of items) {
		result.push({ folder: f, level });
		result.push(...flattenFolders(folders, f.id, level + 1));
	}
	return result;
}

async function handleAddToFolder() {
	if (!selectedFolderId || !id) return;
	try {
		await addDocumentToFolder(id, selectedFolderId);
		toast.success("Documento agregado a la carpeta");
		showFolderDialog = false;
		loadDocumentFolders();
	} catch (err: any) {
		if (err.message?.includes("duplicate") || err.message?.includes("unique")) {
			toast.error("El documento ya está en esta carpeta");
		} else {
			toast.error(`Error: ${err}`);
		}
	}
}

async function handleCreateAndAdd() {
	if (!newFolderName.trim() || !id) return;
	try {
		const folder = await createFolder(
			newFolderName.trim(),
			newFolderParentId || undefined,
		);
		await addDocumentToFolder(id, folder.id);
		toast.success("Carpeta creada y documento agregado");
		showFolderDialog = false;
		loadDocumentFolders();
	} catch (err) {
		toast.error(`Error: ${err}`);
	}
}
```

Add the folder badges section inside the sidebar `Card.Content`, after the "Notas" section (which already has a Separator before it). Insert it after the `status` div and before the `notes` section:

Find this in the template:
```svelte
						{#if doc.notes}
							<Separator />
							...
```

Insert BEFORE that block:

```svelte
						<Separator />
						<div class="flex flex-col gap-2">
							<div class="flex items-center justify-between">
								<span class="text-muted-foreground">Carpetas</span>
								<Button variant="ghost" size="icon-xs" onclick={openFolderDialog}>
									<PlusIcon class="size-3.5" />
								</Button>
							</div>
							{#if docFolders.length === 0}
								<span class="text-xs text-muted-foreground">Sin carpetas</span>
							{:else}
								<div class="flex flex-wrap gap-1">
									{#each docFolders as folder (folder.id)}
										<Badge variant="secondary" class="flex items-center gap-1 cursor-default">
											<FolderIcon class="size-3" />
											<span class="max-w-[120px] truncate">{folder.name}</span>
											<button
												aria-label="Remover de carpeta"
												class="hover:text-destructive ml-0.5"
												onclick={() => handleRemoveFolder(folder.id)}
											>
												<XIcon class="size-3" />
											</button>
										</Badge>
									{/each}
								</div>
							{/if}
						</div>
```

Add the folder picker dialog at the end of the file (before the final closing `</div>`):

```svelte
<!-- Add to Folder Dialog -->
<Dialog.Root open={showFolderDialog} onOpenChange={(o) => { showFolderDialog = o; if (!o) { creatingFolder = false; } }}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Agregar a carpeta</Dialog.Title>
			<Dialog.Description>
				Selecciona una carpeta para "{doc?.name}"
			</Dialog.Description>
		</Dialog.Header>
		<div class="flex flex-col gap-4 py-4">
			<div class="relative">
				<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
				<Input
					bind:value={folderDialogQuery}
					placeholder="Buscar carpetas..."
					class="pl-9"
				/>
			</div>

			<div class="max-h-60 overflow-y-auto flex flex-col gap-1">
				{#each flattenFolders(visibleDialogFolders) as { folder, level } (folder.id)}
					{@const isAlreadyIn = docFolders.some((f) => f.id === folder.id)}
					<button
						type="button"
						class="hover:bg-muted/50 data-[active=true]:bg-muted/80 flex w-full items-center justify-between gap-3 rounded-3xl px-3 py-2 text-left text-sm"
						data-active={selectedFolderId === folder.id}
						style="padding-left: {12 + level * 16}px"
						disabled={isAlreadyIn}
						onclick={() => { selectedFolderId = folder.id; }}
					>
						<div class="flex min-w-0 items-center gap-2">
							<FolderIcon class="text-muted-foreground size-4 shrink-0" />
							<span class="truncate">{folder.name}</span>
						</div>
						{#if isAlreadyIn}
							<CheckIcon class="text-muted-foreground size-4 shrink-0" />
						{/if}
					</button>
				{/each}
				{#if visibleDialogFolders.length === 0}
					<p class="text-muted-foreground py-4 text-center text-sm">No se encontraron carpetas</p>
				{/if}
			</div>

			<Separator />

			{#if !creatingFolder}
				<Button variant="ghost" size="sm" class="justify-start" onclick={() => { creatingFolder = true; }}>
					<PlusIcon class="mr-2 size-4" />
					Crear nueva carpeta
				</Button>
			{:else}
				<div class="flex flex-col gap-3 rounded-2xl border p-3">
					<div class="flex flex-col gap-2">
						<Label for="new-folder-name">Nombre</Label>
						<Input id="new-folder-name" bind:value={newFolderName} placeholder="Nombre de la carpeta" />
					</div>
					<div class="flex flex-col gap-2">
						<Label for="new-folder-parent">Carpeta padre (opcional)</Label>
						<select
							id="new-folder-parent"
							bind:value={newFolderParentId}
							class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-10 w-full rounded-2xl border px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
						>
							<option value="">Ninguna (raíz)</option>
							{#each folderList as f (f.id)}
								<option value={f.id}>{f.name}</option>
							{/each}
						</select>
					</div>
					<div class="flex gap-2 justify-end">
						<Button variant="outline" size="sm" onclick={() => { creatingFolder = false; }}>
							Cancelar
						</Button>
						<Button size="sm" onclick={handleCreateAndAdd} disabled={!newFolderName.trim()}>
							Crear y agregar
						</Button>
					</div>
				</div>
			{/if}
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => { showFolderDialog = false; }}>
				Cancelar
			</Button>
			<Button onclick={handleAddToFolder} disabled={!selectedFolderId}>
				Agregar
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
```

- [ ] **Step 2: Run svelte-check**

Run: `npx svelte-check` in `web/` directory.

- [ ] **Step 3: Commit**

```bash
git add web/src/routes/document/[id]/+page.svelte
git commit -m "feat: add folder badges and 'Add to folder' dialog to document page"
```
