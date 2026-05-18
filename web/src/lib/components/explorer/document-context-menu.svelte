<script lang="ts">
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";
	import type { Snippet } from "svelte";
	import StarIcon from "@lucide/svelte/icons/star";
	import StarOffIcon from "@lucide/svelte/icons/star-off";
	import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
	import CopyIcon from "@lucide/svelte/icons/copy";
	import FolderPlusIcon from "@lucide/svelte/icons/folder-plus";
	import FolderMinusIcon from "@lucide/svelte/icons/folder-minus";
	import FolderIcon from "@lucide/svelte/icons/folder";
	import CheckIcon from "@lucide/svelte/icons/check";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import SearchIcon from "@lucide/svelte/icons/search";
	import * as ContextMenu from "$lib/components/ui/context-menu/index.js";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Label } from "$lib/components/ui/label/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import {
		toggleFavorite,
		updateDocumentCategory,
		deleteDocument,
		getCategoriesWithSubcategories,
		openDocument,
		addDocumentToFolder,
		removeDocumentFromFolder,
		getFolders,
		createFolder,
	} from "$lib/api";
	import type { ExplorerFolder } from "$lib/types";

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

	let categories = $state<Awaited<ReturnType<typeof getCategoriesWithSubcategories>>>([]);

	let showFolderDialog = $state(false);
	let folderDialogQuery = $state("");
	let folderList = $state<ExplorerFolder[]>([]);
	let selectedFolderId = $state<string | "">("");
	let creatingFolder = $state(false);
	let newFolderName = $state("");
	let newFolderParentId = $state<string | "">("");

	$effect(() => {
		getCategoriesWithSubcategories().then((cats) => { categories = cats; }).catch(() => {});
	});

	async function handleToggleFavorite() {
		try {
			const newVal = await toggleFavorite(docId, isFavorite);
			isFavorite = newVal;
			toast.success(newVal ? "Agregado a favoritos" : "Eliminado de favoritos");
			if (onFavoriteToggle) {
				onFavoriteToggle(newVal);
			} else {
				onchange();
			}
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleMoveToCategory(categoryId: string, subcategoryId: string) {
		try {
			await updateDocumentCategory(docId, categoryId, subcategoryId);
			toast.success("Documento movido");
			onchange();
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleDelete() {
		if (!confirm(`¿Eliminar "${docName}" permanentemente?`)) return;
		try {
			await deleteDocument(docId);
			toast.success("Documento eliminado");
			onchange();
		} catch (err) {
			toast.error(`Error al eliminar: ${err}`);
		}
	}

	async function handleOpenExternally() {
		try {
			await openDocument(docId);
			toast.info("Abriendo documento…");
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleCopyPath() {
		const base = (await import("$lib/pocketbase")).default.baseUrl.replace(/\/$/, "");
		const url = `${base}/api/documents/open/${docId}`;
		await navigator.clipboard.writeText(url);
		toast.success("Ruta copiada");
	}

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

	function flattenFolders(folders: ExplorerFolder[], parentId: string | null = null, level = 0): { folder: ExplorerFolder; level: number }[] {
		const result: { folder: ExplorerFolder; level: number }[] = [];
		const items = folders.filter((f) => (parentId === null ? !f.parentId : f.parentId === parentId));
		for (const f of items) {
			result.push({ folder: f, level });
			result.push(...flattenFolders(folders, f.id, level + 1));
		}
		return result;
	}

	let docFolderIds = $state<string[]>([]);

	async function loadDocFolderIds() {
		try {
			const folders = await (await import("$lib/api")).getFoldersForDocument(docId);
			docFolderIds = folders.map((f) => f.id);
		} catch {
			docFolderIds = [];
		}
	}

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
</script>

<ContextMenu.Root>
	<ContextMenu.Trigger class="block">
		{@render children?.()}
	</ContextMenu.Trigger>
	<ContextMenu.Content class="min-w-52">
		{#if isVisualizable}
			<ContextMenu.Item onclick={() => goto(`/document/${docId}`)}>
				<FolderOpenIcon class="mr-2 size-4" />
				Abrir documento
			</ContextMenu.Item>
			<ContextMenu.Separator />
		{/if}
		<ContextMenu.Item onclick={handleToggleFavorite}>
			{#if isFavorite}
				<StarOffIcon class="mr-2 size-4" />
				Quitar de favoritos
			{:else}
				<StarIcon class="mr-2 size-4" />
				Agregar a favoritos
			{/if}
		</ContextMenu.Item>
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
		<ContextMenu.Item onclick={handleOpenExternally}>
			<ExternalLinkIcon class="mr-2 size-4" />
			Abrir externamente
		</ContextMenu.Item>
		{#if categories.length > 0}
			<ContextMenu.Separator />
			<ContextMenu.Sub>
				<ContextMenu.SubTrigger class="flex items-center gap-2">
					<FolderOpenIcon class="size-4" />
					Mover a categoría
					<ChevronRightIcon class="ml-auto size-4" />
				</ContextMenu.SubTrigger>
				<ContextMenu.SubContent class="min-w-48">
					{#each categories as cat (cat.id)}
						<ContextMenu.Sub>
							<ContextMenu.SubTrigger class="flex items-center gap-2">
								<span style="background-color: {cat.color}" class="size-2 rounded-full shrink-0"></span>
								{cat.name}
								<ChevronRightIcon class="ml-auto size-3" />
							</ContextMenu.SubTrigger>
							<ContextMenu.SubContent class="min-w-40">
								{#each cat.subcategories as sub (sub.id)}
									<ContextMenu.Item onclick={() => handleMoveToCategory(cat.id, sub.id)}>
										{sub.name}
									</ContextMenu.Item>
								{/each}
							</ContextMenu.SubContent>
						</ContextMenu.Sub>
					{/each}
				</ContextMenu.SubContent>
			</ContextMenu.Sub>
		{/if}
		<ContextMenu.Separator />
		<ContextMenu.Item onclick={handleCopyPath}>
			<CopyIcon class="mr-2 size-4" />
			Copiar ruta
		</ContextMenu.Item>
		<ContextMenu.Separator />
		<ContextMenu.Item class="text-destructive" onclick={handleDelete}>
			<TrashIcon class="mr-2 size-4" />
			Eliminar
		</ContextMenu.Item>
	</ContextMenu.Content>
</ContextMenu.Root>

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
