<script lang="ts">
	import SearchIcon from "@lucide/svelte/icons/search";
	import FolderPlusIcon from "@lucide/svelte/icons/folder-plus";
	import UploadIcon from "@lucide/svelte/icons/upload";
	import LayoutGridIcon from "@lucide/svelte/icons/layout-grid";
	import ListIcon from "@lucide/svelte/icons/list";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";

	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { ButtonGroup } from "$lib/components/ui/button-group/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Empty } from "$lib/components/ui/empty/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Item } from "$lib/components/ui/item/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";

	import {
		FOLDERS,
		FolderIcon,
		formatDate,
		type ExplorerFolder,
	} from "$lib/mock/explorer";

	type ViewMode = "grid" | "list";

	let view: ViewMode = $state("grid");
	let query = $state("");
	let selectedFolderId: string | null = $state(FOLDERS[0]?.id ?? null);

	const visibleFolders = $derived.by(() => {
		const q = query.trim().toLowerCase();
		return FOLDERS.filter((f) => !q || f.name.toLowerCase().includes(q));
	});

	const selectedFolder: ExplorerFolder | null = $derived.by(() => {
		if (!selectedFolderId) return null;
		return FOLDERS.find((f) => f.id === selectedFolderId) ?? null;
	});
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="min-w-0">
			<h1 class="text-2xl font-semibold leading-tight">Carpetas</h1>
			<p class="text-muted-foreground">
				Agrupa y navega como explorer (estilo Drive). Después conectamos la lógica.
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<Button variant="outline">
				<FolderPlusIcon class="mr-2 size-4" />
				Nueva carpeta
			</Button>
			<Button>
				<UploadIcon class="mr-2 size-4" />
				Subir
			</Button>
		</div>
	</div>

	{#if FOLDERS.length === 0}
		<Empty>
			<div class="text-muted-foreground">
				Aún no tienes carpetas. Crea una para agrupar documentos.
			</div>
			<Button>
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
					<Card.Description>Lista y búsqueda rápida.</Card.Description>
				</Card.Header>
				<Card.Content class="pt-0">
					<div class="flex flex-col gap-3">
						<div class="relative">
							<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
							<Input bind:value={query} placeholder="Buscar carpetas…" class="pl-9" />
						</div>

						<div class="flex items-center justify-between text-sm">
							<span class="text-muted-foreground">{visibleFolders.length} carpeta(s)</span>
							<Badge variant="secondary">Agrupar</Badge>
						</div>

						<Separator />

						<div class="flex flex-col gap-1">
							{#each visibleFolders as folder (folder.id)}
								<button
									type="button"
									class="hover:bg-muted/50 focus-visible:ring-ring/30 flex w-full items-center justify-between gap-3 rounded-3xl px-3 py-2 text-left text-sm outline-none focus-visible:ring-3"
									data-active={selectedFolderId === folder.id}
									onclick={() => (selectedFolderId = folder.id)}
								>
									<div class="flex min-w-0 items-center gap-2">
										<FolderIcon class="text-muted-foreground size-4" />
										<span class="truncate font-medium">{folder.name}</span>
									</div>
									<Badge variant="secondary">{folder.count}</Badge>
								</button>
							{/each}
						</div>
					</div>
				</Card.Content>
			</Card.Root>

			<div class="flex flex-col gap-4">
				<div class="flex flex-wrap items-center gap-2">
					<div class="relative min-w-[260px] flex-1">
						<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
						<Input placeholder="Buscar dentro (próximamente)..." class="pl-9" />
					</div>

					<ButtonGroup class="ml-auto">
						<Button
							variant="outline"
							size="icon-sm"
							aria-label="Vista grid"
							data-active={view === "grid"}
							onclick={() => (view = "grid")}
						>
							<LayoutGridIcon class="size-4" />
						</Button>
						<Button
							variant="outline"
							size="icon-sm"
							aria-label="Vista lista"
							data-active={view === "list"}
							onclick={() => (view = "list")}
						>
							<ListIcon class="size-4" />
						</Button>
					</ButtonGroup>
				</div>

				<div class="text-muted-foreground flex flex-wrap items-center gap-2 text-sm">
					<Badge variant="secondary">
						{selectedFolder?.name ?? "Selecciona una carpeta"}
					</Badge>
					{#if selectedFolder}
						<span class="inline-flex items-center gap-1">
							<ChevronRightIcon class="size-4" />
							<span>{selectedFolder.count} item(s)</span>
						</span>
					{/if}
				</div>

				{#if !selectedFolder}
					<Empty>
						<div class="text-muted-foreground">Selecciona una carpeta para ver su contenido.</div>
					</Empty>
				{:else if view === "grid"}
					<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
						{#each visibleFolders.filter((f) => f.parentId === selectedFolder.id) as folder (folder.id)}
							<Card.Root class="shadow-sm">
								<Card.Header class="pb-2">
									<div class="flex items-start justify-between gap-3">
										<div class="min-w-0">
											<Card.Title class="flex items-center gap-2">
												<FolderIcon class="text-muted-foreground size-4" />
												<span class="truncate">{folder.name}</span>
											</Card.Title>
											<Card.Description class="truncate">
												{folder.count} item(s) · {formatDate(folder.updatedAt)}
											</Card.Description>
										</div>
										<Button variant="ghost" size="sm">Abrir</Button>
									</div>
								</Card.Header>
							</Card.Root>
						{/each}

						{#if visibleFolders.filter((f) => f.parentId === selectedFolder.id).length === 0}
							<Empty class="sm:col-span-2 xl:col-span-3">
								<div class="text-muted-foreground">
									No hay subcarpetas aquí (todavía). Luego mostraremos documentos dentro.
								</div>
								<Button variant="outline">
									<FolderPlusIcon class="mr-2 size-4" />
									Crear subcarpeta
								</Button>
							</Empty>
						{/if}
					</div>
				{:else}
					<div class="flex flex-col">
						{#each visibleFolders.filter((f) => f.parentId === selectedFolder.id) as folder (folder.id)}
							<Item variant="muted" size="sm" class="rounded-3xl">
								<div class="flex w-full items-center gap-3">
									<div class="bg-muted flex size-9 items-center justify-center rounded-2xl">
										<FolderIcon class="text-muted-foreground size-4" />
									</div>
									<div class="min-w-0 flex-1">
										<div class="truncate font-medium">{folder.name}</div>
										<div class="text-muted-foreground truncate text-xs">
											{folder.count} item(s) · {formatDate(folder.updatedAt)}
										</div>
									</div>
									<Button variant="ghost" size="sm">Abrir</Button>
								</div>
							</Item>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

