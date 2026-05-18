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
