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

	const rootFolders = $derived.by(() => {
		return visibleFolders.filter((f) => !f.parentId);
	});

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
