<script lang="ts">
	import { onMount } from "svelte";
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import FolderKanbanIcon from "@lucide/svelte/icons/folder-kanban";
	import SearchIcon from "@lucide/svelte/icons/search";
	import LayoutGridIcon from "@lucide/svelte/icons/layout-grid";
	import ListIcon from "@lucide/svelte/icons/list";
	import XIcon from "@lucide/svelte/icons/x";

	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { ButtonGroup } from "$lib/components/ui/button-group/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Item } from "$lib/components/ui/item/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";

	import FileCard from "$lib/components/explorer/file-card.svelte";
	import DocumentContextMenu from "$lib/components/explorer/document-context-menu.svelte";

	import { fileIconFor, formatBytes, formatDate, type ExplorerFile } from "$lib/types";
	import { getCategories, getDocumentsByCategory, type DocumentDetail } from "$lib/api";

	let categoryId = $state("");
	let categoryName = $state("");
	let categoryColor = $state("#71717a");
	let categoryDesc = $state("");
	let categoryCount = $state(0);
	let subcategories = $state<{ id: string; name: string }[]>([]);

	let documents = $state<ExplorerFile[]>([]);
	let selectedSub = $state<string | "all">("all");
	let query = $state("");
	let debouncedQuery = $state("");
	let view: "grid" | "list" = $state("grid");
	let loading = $state(true);

	$effect(() => {
		const val = query;
		const timer = setTimeout(() => { debouncedQuery = val; }, 250);
		return () => clearTimeout(timer);
	});

	onMount(async () => {
		categoryId = $page.params.id ?? "";
		if (!categoryId) return;
		await loadData();
	});

	async function loadData() {
		loading = true;
		try {
			const [cats, docs] = await Promise.all([
				getCategories(),
				getDocumentsByCategory(categoryId),
			]);
			const cat = cats.find((c) => c.id === categoryId);
			if (cat) {
				categoryName = cat.name;
				categoryColor = cat.color;
				categoryCount = cat.count;
				subcategories = cat.subcategories;
			}
			documents = docs;
		} catch (err) {
			console.error("Failed to load category:", err);
		} finally {
			loading = false;
		}
	}

	const filteredDocs = $derived.by(() => {
		const q = debouncedQuery.trim().toLowerCase();
		let docs = documents;
		if (selectedSub !== "all") {
			docs = docs.filter((d) => d.subcategory_id === selectedSub);
		}
		if (q) {
			docs = docs.filter((d) =>
				d.name.toLowerCase().includes(q) ||
				(d.locationLabel ?? "").toLowerCase().includes(q)
			);
		}
		return docs;
	});
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center gap-2">
		<Button variant="ghost" size="icon-sm" onclick={() => goto("/documents")}>
			<ArrowLeftIcon class="size-4" />
		</Button>
		<div class="flex items-center gap-3">
			<span style="background-color: {categoryColor}" class="size-4 rounded-full inline-block"></span>
			<h1 class="text-2xl font-semibold leading-tight">{categoryName}</h1>
		</div>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-10 text-center">Cargando…</div>
	{:else}
		<div class="grid gap-6 lg:grid-cols-[280px_1fr]">
			<div class="flex flex-col gap-4">
				<Card.Root class="shadow-sm">
					<Card.Header class="pb-2">
						<Card.Title class="flex items-center gap-2">
							<FolderKanbanIcon class="text-muted-foreground size-4" />
							Subcategorías
						</Card.Title>
						<Card.Description>{subcategories.length} subcategorías · {categoryCount} documentos</Card.Description>
					</Card.Header>
					<Card.Content class="pt-0">
						<div class="flex flex-col gap-1">
							<button
								type="button"
								class="hover:bg-muted/50 flex w-full items-center justify-between gap-3 rounded-3xl px-3 py-2 text-left text-sm"
								data-active={selectedSub === "all"}
								onclick={() => (selectedSub = "all")}
							>
								<span class="font-medium">Todas</span>
								<Badge variant="secondary">{documents.length}</Badge>
							</button>
							<Separator class="my-1" />
							{#each subcategories as sub (sub.id)}
								<button
									type="button"
									class="hover:bg-muted/50 flex w-full items-center justify-between gap-3 rounded-3xl px-3 py-2 text-left text-sm"
									data-active={selectedSub === sub.id}
									onclick={() => (selectedSub = sub.id)}
								>
									<span class="truncate">{sub.name}</span>
										<Badge variant="secondary">
											{documents.filter((d) => d.subcategory_id === sub.id).length}
										</Badge>
								</button>
							{/each}
						</div>
					</Card.Content>
				</Card.Root>
			</div>

			<div class="flex flex-col gap-4">
				<div class="flex flex-wrap items-center gap-2">
					<div class="relative min-w-[260px] flex-1">
						<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
						<Input bind:value={query} placeholder="Buscar documentos en esta categoría…" class="pl-9" />
						{#if query}
							<button
								class="absolute right-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground hover:text-foreground"
								onclick={() => (query = "")}
							>
								<XIcon class="size-4" />
							</button>
						{/if}
					</div>
					<ButtonGroup>
						<Button variant="outline" size="icon-sm" aria-label="Grid" data-active={view === "grid"} onclick={() => (view = "grid")}>
							<LayoutGridIcon class="size-4" />
						</Button>
						<Button variant="outline" size="icon-sm" aria-label="Lista" data-active={view === "list"} onclick={() => (view = "list")}>
							<ListIcon class="size-4" />
						</Button>
					</ButtonGroup>
				</div>

				<div class="flex flex-wrap items-center gap-2">
					<Badge variant="secondary">
						{selectedSub === "all" ? "Todas las subcategorías" : (subcategories.find((s) => s.id === selectedSub)?.name ?? "")}
					</Badge>
					<span class="text-muted-foreground text-sm">{filteredDocs.length} documento(s)</span>
				</div>

				{#if filteredDocs.length === 0}
					<div class="text-muted-foreground py-10 text-center">
						{#if query || selectedSub !== "all"}
							Sin resultados con los filtros actuales.
						{:else}
							No hay documentos en esta categoría.
						{/if}
					</div>
				{:else if view === "grid"}
					<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
						{#each filteredDocs as file (file.id)}
							<FileCard {file} href={`/document/${file.id}`} />
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
								isVisualizable={!["docx", "xlsx", "doc", "xls", "ppt", "pptx", "odt", "ods", "odp", "zip"].includes(file.ext)}
								onchange={() => {}}
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
