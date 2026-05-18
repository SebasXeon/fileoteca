<script lang="ts">
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import { onMount } from "svelte";
	import { toast } from "svelte-sonner";
	import FileTextIcon from "@lucide/svelte/icons/file-text";
	import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import StarIcon from "@lucide/svelte/icons/star";
	import StarOffIcon from "@lucide/svelte/icons/star-off";
	import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import MoreHorizontalIcon from "@lucide/svelte/icons/ellipsis";
	import XIcon from "@lucide/svelte/icons/x";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import SearchIcon from "@lucide/svelte/icons/search";
	import CheckIcon from "@lucide/svelte/icons/check";
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Label } from "$lib/components/ui/label/index.js";
	import {
		getDocument,
		openDocument,
		getOpenUrl,
		toggleFavorite,
		updateDocumentCategory,
		deleteDocument,
		getCategoriesWithSubcategories,
		getFoldersForDocument,
		removeDocumentFromFolder,
		addDocumentToFolder,
		getFolders,
		createFolder,
		type DocumentDetail,
	} from "$lib/api";
	import { FolderIcon, formatBytes, formatDate, type ExplorerFolder } from "$lib/types";


	let doc = $state<DocumentDetail | null>(null);
	let loading = $state(true);
	let error = $state("");
	let opening = $state(false);
	let opened = $state(false);
	let openCooldown = $state(false);
	let textContent = $state("");
	let docFolders = $state<{ id: string; name: string }[]>([]);
	let showFolderDialog = $state(false);
	let folderDialogQuery = $state("");
	let folderList = $state<ExplorerFolder[]>([]);
	let selectedFolderId = $state<string | "">("");
	let creatingFolder = $state(false);
	let newFolderName = $state("");
	let newFolderParentId = $state<string | "">("");
	let categories = $state<Awaited<ReturnType<typeof getCategoriesWithSubcategories>>>([]);

	const id = $derived($page.params.id);

	const inlineExts = [
		"pdf", "png", "jpg", "jpeg", "gif", "bmp", "svg", "webp", "tiff", "ico",
		"txt", "csv", "md", "json", "xml", "html", "htm", "rtf",
	];

	const isVisualizable = $derived.by(() => {
		if (!doc) return false;
		return inlineExts.includes(doc.ext);
	});

	const isImage = $derived.by(() => {
		if (!doc) return false;
		return ["png", "jpg", "jpeg", "gif", "bmp", "svg", "webp", "tiff", "ico"].includes(doc.ext);
	});

	const isText = $derived.by(() => {
		if (!doc) return false;
		return inlineExts.includes(doc.ext) && !isImage && doc.ext !== "pdf";
	});

	const openUrl = $derived.by(() => {
		if (!id) return "";
		return getOpenUrl(id);
	});

	onMount(async () => {
		await loadDocument();
		getCategoriesWithSubcategories().then((cats) => { categories = cats; }).catch(() => {});
		loadDocumentFolders();
	});

	async function loadDocument() {
		try {
			if (!id) throw new Error("ID de documento no proporcionado");
			loading = true;
			const d = await getDocument(id);
			doc = d;
			if (isText) {
				const resp = await fetch(openUrl);
				if (!resp.ok) throw new Error(`No se pudo cargar el contenido: ${resp.status}`);
				textContent = await resp.text();
			}
		} catch (err) {
			error = String(err);
		} finally {
			loading = false;
		}
	}

	async function handleOpenExternally() {
		if (!doc || opening || openCooldown) return;
		opening = true;
		error = "";
		try {
			await openDocument(doc.id);
			opened = true;
			openCooldown = true;
			setTimeout(() => { openCooldown = false; }, 2000);
		} catch (err) {
			error = String(err);
		} finally {
			opening = false;
		}
	}

	async function handleToggleFavorite() {
		if (!doc) return;
		try {
			const newVal = await toggleFavorite(doc.id, doc.favorite ?? false);
			doc = { ...doc, favorite: newVal };
			toast.success(newVal ? "Agregado a favoritos" : "Eliminado de favoritos");
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleMoveToCategory(categoryId: string, subcategoryId: string) {
		if (!doc || !id) return;
		try {
			await updateDocumentCategory(id, categoryId, subcategoryId);
			toast.success("Documento movido");
			await loadDocument();
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleDelete() {
		if (!doc || !id) return;
		if (!confirm(`¿Eliminar "${doc.name}" permanentemente?`)) return;
		try {
			await deleteDocument(id);
			toast.success("Documento eliminado");
			setTimeout(() => goto("/"), 500);
		} catch (err) {
			toast.error(`Error al eliminar: ${err}`);
		}
	}

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
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center gap-2">
		<Button variant="ghost" size="sm" onclick={() => goto("/")}>
			<ArrowLeftIcon class="mr-1 size-4" />
			Volver
		</Button>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-10 text-center">Cargando documento…</div>
	{:else if error}
		<div class="text-destructive py-10 text-center">Error: {error}</div>
	{:else if doc}
		<div class="flex flex-col gap-6">
			<div class="flex flex-wrap items-start justify-between gap-4">
				<div class="min-w-0">
					<h1 class="text-2xl font-semibold leading-tight">{doc.name}</h1>
					<p class="text-muted-foreground">{doc.locationLabel}</p>
				</div>
<div class="flex items-center gap-2">
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								<Button variant="outline" {...props}>
									<MoreHorizontalIcon class="size-4" />
								</Button>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="end" class="min-w-52">
							<DropdownMenu.Item onclick={handleToggleFavorite}>
								{#if doc.favorite}
									<StarOffIcon class="mr-2 size-4" />
									Quitar de favoritos
								{:else}
									<StarIcon class="mr-2 size-4" />
									Agregar a favoritos
								{/if}
							</DropdownMenu.Item>
							<DropdownMenu.Item onclick={handleOpenExternally} disabled={opening || openCooldown}>
								<ExternalLinkIcon class="mr-2 size-4" />
								Abrir externamente
							</DropdownMenu.Item>
							{#if categories.length > 0}
								<DropdownMenu.Separator />
								<DropdownMenu.Sub>
									<DropdownMenu.SubTrigger class="flex items-center gap-2">
										<FolderOpenIcon class="size-4" />
										Mover a categoría
										<ChevronRightIcon class="ml-auto size-4" />
									</DropdownMenu.SubTrigger>
									<DropdownMenu.SubContent class="min-w-48">
										{#each categories as cat (cat.id)}
											<DropdownMenu.Sub>
												<DropdownMenu.SubTrigger class="flex items-center gap-2">
													<span style="background-color: {cat.color}" class="size-2 rounded-full shrink-0"></span>
													{cat.name}
													<ChevronRightIcon class="ml-auto size-3" />
												</DropdownMenu.SubTrigger>
												<DropdownMenu.SubContent class="min-w-40">
													{#each cat.subcategories as sub (sub.id)}
														<DropdownMenu.Item onclick={() => handleMoveToCategory(cat.id, sub.id)} disabled={doc.category_id === cat.id && doc.subcategory_id === sub.id}>
															{sub.name}
														</DropdownMenu.Item>
													{/each}
												</DropdownMenu.SubContent>
											</DropdownMenu.Sub>
										{/each}
									</DropdownMenu.SubContent>
								</DropdownMenu.Sub>
							{/if}
							<DropdownMenu.Separator />
							<DropdownMenu.Item class="text-destructive" onclick={handleDelete}>
								<TrashIcon class="mr-2 size-4" />
								Eliminar documento
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</div>
			</div>

			<div class="grid gap-6 lg:grid-cols-[320px_1fr]">
				<Card.Root class="shadow-sm h-fit">
					<Card.Header>
						<Card.Title>Información</Card.Title>
					</Card.Header>
					<Card.Content class="flex flex-col gap-3 text-sm">
						{#if doc.thumbnail}
							<div class="rounded-lg overflow-hidden bg-muted">
								<img src={doc.thumbnail} alt={doc.name} class="w-full h-auto object-cover max-h-48" />
							</div>
						{/if}
						<div class="flex justify-between">
							<span class="text-muted-foreground">Nombre</span>
							<span class="truncate max-w-[180px]">{doc.name}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">Extensión</span>
							<Badge variant="secondary">{doc.ext.toUpperCase()}</Badge>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">Tamaño</span>
							<span>{formatBytes(doc.sizeBytes)}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">Actualizado</span>
							<span>{formatDate(doc.updatedAt)}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">Categoría</span>
							<span class="truncate max-w-[180px]">{doc.category || "—"}</span>
						</div>
						{#if doc.source_type}
							<div class="flex justify-between">
								<span class="text-muted-foreground">Origen</span>
								<span class="capitalize">{doc.source_type.replace("_", " ")}</span>
							</div>
						{/if}
						{#if doc.status}
							<div class="flex justify-between">
								<span class="text-muted-foreground">Estado</span>
								<Badge variant="outline" class="capitalize">{doc.status}</Badge>
							</div>
						{/if}
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
						{#if doc.notes}
							<Separator />
							<div class="flex flex-col gap-1">
								<span class="text-muted-foreground">Notas</span>
								<p class="text-xs whitespace-pre-wrap">{doc.notes}</p>
							</div>
						{/if}
					</Card.Content>
				</Card.Root>

				<div class="flex flex-col gap-4">
					{#if isVisualizable}
						<Card.Root class="shadow-sm overflow-hidden">
							{#if doc.ext === "pdf"}
								<embed src={openUrl} type="application/pdf" class="w-full h-[70vh] border-0" />
							{:else if isImage}
								<img
									src={openUrl}
									alt={doc.name}
									class="max-w-full h-auto max-h-[70vh] object-contain mx-auto"
								/>
							{:else if isText}
								<div class="p-4 bg-muted/30">
									<pre class="text-xs overflow-auto max-h-[70vh] whitespace-pre-wrap">{textContent}</pre>
								</div>
							{/if}
						</Card.Root>
					{:else}
						<Card.Root class="shadow-sm">
							<Card.Content class="py-12 text-center flex flex-col items-center gap-4">
								{#if opening}
									<FileTextIcon class="text-muted-foreground size-12 animate-pulse" />
									<p class="text-lg font-medium">Abriendo documento…</p>
									<p class="text-muted-foreground text-sm">Se está abriendo en su aplicación predeterminada</p>
								{:else if opened}
									<FileTextIcon class="text-muted-foreground size-12" />
									<p class="text-lg font-medium">Documento abierto</p>
									<p class="text-muted-foreground text-sm">El archivo se abrió en su aplicación predeterminada</p>
								{:else}
									<FileTextIcon class="text-muted-foreground size-12" />
									<div class="flex flex-col gap-2">
										<p class="text-lg font-medium">Este documento no se puede previsualizar</p>
										<p class="text-muted-foreground text-sm">Se abrirá con el programa correspondiente en tu equipo</p>
									</div>
									<Button size="lg" onclick={handleOpenExternally} disabled={openCooldown}>
										<ExternalLinkIcon class="mr-2 size-4" />
										Abrir documento
									</Button>
								{/if}
							</Card.Content>
						</Card.Root>
					{/if}
				</div>
			</div>
		</div>
	{:else}
		<div class="text-muted-foreground py-10 text-center">Documento no encontrado</div>
	{/if}
</div>

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
