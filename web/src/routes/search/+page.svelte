<script lang="ts">
	import { onMount } from "svelte";
	import { page } from "$app/stores";
	import { goto } from "$app/navigation";
	import SearchIcon from "@lucide/svelte/icons/search";
	import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
	import SlidersHorizontalIcon from "@lucide/svelte/icons/sliders-horizontal";
	import ArrowUpDownIcon from "@lucide/svelte/icons/arrow-up-down";
	import LayoutGridIcon from "@lucide/svelte/icons/layout-grid";
	import ListIcon from "@lucide/svelte/icons/list";
	import XIcon from "@lucide/svelte/icons/x";

	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { ButtonGroup } from "$lib/components/ui/button-group/index.js";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import { Item } from "$lib/components/ui/item/index.js";

	import FileCard from "$lib/components/explorer/file-card.svelte";
	import DocumentContextMenu from "$lib/components/explorer/document-context-menu.svelte";

	import { fileIconFor, formatBytes, formatDate, type ExplorerFile } from "$lib/types";
	import { searchDocuments } from "$lib/api";

	type ViewMode = "grid" | "list";
	type SortMode = "recent" | "name" | "size";
	type TypeFilter = "all" | "pdf" | "docx" | "xlsx" | "image";

	let query = $state("");
	let debouncedQuery = $state("");
	let view: ViewMode = $state("grid");
	let sort: SortMode = $state("recent");
	let typeFilter: TypeFilter = $state("all");

	let results = $state<ExplorerFile[]>([]);
	let loading = $state(false);
	let error = $state("");
	let searched = $state(false);
	let initialLoad = $state(true);

	$effect(() => {
		const val = query;
		const timer = setTimeout(() => {
			debouncedQuery = val;
		}, 300);
		return () => clearTimeout(timer);
	});

	$effect(() => {
		if (initialLoad) return;
		const trimmed = debouncedQuery.trim();
		if (trimmed) {
			const url = new URL($page.url);
			url.searchParams.set("q", trimmed);
			goto(url.toString(), { replaceState: true, noScroll: true });
			doSearch(trimmed);
		} else {
			const url = new URL($page.url);
			url.searchParams.delete("q");
			goto(url.toString(), { replaceState: true, noScroll: true });
			results = [];
			searched = false;
		}
	});

	onMount(() => {
		const q = $page.url.searchParams.get("q") ?? "";
		if (q) {
			query = q;
			debouncedQuery = q;
			doSearch(q);
		}
		initialLoad = false;
	});

	async function doSearch(q: string) {
		const trimmed = q.trim();
		if (!trimmed) {
			results = [];
			searched = false;
			return;
		}
		loading = true;
		error = "";
		searched = true;
		try {
			results = await searchDocuments(trimmed);
		} catch (err) {
			error = String(err);
		} finally {
			loading = false;
		}
	}

	function clearSearch() {
		query = "";
		results = [];
		searched = false;
		goto("/search", { replaceState: true });
	}

	function matchesType(file: ExplorerFile) {
		if (typeFilter === "all") return true;
		if (typeFilter === "image") return file.ext === "png" || file.ext === "jpg";
		return file.ext === typeFilter;
	}

	function sortFiles(files: ExplorerFile[]) {
		const list = [...files];
		list.sort((a, b) => {
			if (sort === "name") return a.name.localeCompare(b.name, "es");
			if (sort === "size") return b.sizeBytes - a.sizeBytes;
			return b.updatedAt.getTime() - a.updatedAt.getTime();
		});
		return list;
	}

	const filtered = $derived.by(() =>
		sortFiles(results.filter(matchesType))
	);
</script>

<svelte:head>
	<title>Búsqueda: {query || "Fileoteca"}</title>
</svelte:head>

<div class="flex flex-col gap-6">
	<div class="flex items-center gap-2">
		<Button variant="ghost" size="icon-sm" onclick={() => goto("/")}>
			<ArrowLeftIcon class="size-4" />
		</Button>
		<h1 class="text-2xl font-semibold leading-tight">Buscar</h1>
	</div>

	<div class="flex flex-col gap-3">
		<div class="flex flex-wrap items-center gap-2">
			<div class="relative min-w-[280px] flex-1">
				<SearchIcon
					class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2"
				/>
				<Input
					bind:value={query}
					placeholder="Buscar por nombre, contenido o texto extraído…"
					class="pl-9"
				/>
				{#if query}
					<button
						class="absolute right-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground hover:text-foreground"
						onclick={clearSearch}
					>
						<XIcon class="size-4" />
					</button>
				{/if}
			</div>

			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button variant="outline" {...props}>
							<SlidersHorizontalIcon class="mr-2 size-4" />
							Tipo
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end" class="min-w-56">
					<DropdownMenu.Label>Filtrar</DropdownMenu.Label>
					<DropdownMenu.Separator />
					<DropdownMenu.RadioGroup bind:value={typeFilter}>
						<DropdownMenu.RadioItem value="all">Todos</DropdownMenu.RadioItem>
						<DropdownMenu.RadioItem value="pdf">PDF</DropdownMenu.RadioItem>
						<DropdownMenu.RadioItem value="docx">Word</DropdownMenu.RadioItem>
						<DropdownMenu.RadioItem value="xlsx">Excel</DropdownMenu.RadioItem>
						<DropdownMenu.RadioItem value="image">Imágenes</DropdownMenu.RadioItem>
					</DropdownMenu.RadioGroup>
				</DropdownMenu.Content>
			</DropdownMenu.Root>

			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button variant="outline" {...props}>
							<ArrowUpDownIcon class="mr-2 size-4" />
							Ordenar
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end" class="min-w-56">
					<DropdownMenu.Label>Orden</DropdownMenu.Label>
					<DropdownMenu.Separator />
					<DropdownMenu.RadioGroup bind:value={sort}>
						<DropdownMenu.RadioItem value="recent">Más recientes</DropdownMenu.RadioItem>
						<DropdownMenu.RadioItem value="name">Nombre</DropdownMenu.RadioItem>
						<DropdownMenu.RadioItem value="size">Tamaño</DropdownMenu.RadioItem>
					</DropdownMenu.RadioGroup>
				</DropdownMenu.Content>
			</DropdownMenu.Root>

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
			{#if searched}
				<span>{filtered.length} resultado(s)</span>
			{/if}
			{#if typeFilter !== "all"}
				<Separator orientation="vertical" class="h-4" />
				<Badge variant="outline">Tipo: {typeFilter}</Badge>
			{/if}
			{#if query.trim()}
				<Separator orientation="vertical" class="h-4" />
				<Badge variant="outline">"{query.trim()}"</Badge>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-10 text-center">Buscando…</div>
	{:else if error}
		<div class="text-destructive py-10 text-center">Error: {error}</div>
	{:else if !searched}
		<div class="text-muted-foreground py-10 text-center flex flex-col items-center gap-3">
			<SearchIcon class="size-12 opacity-30" />
			<p>Busca documentos por nombre, contenido o texto extraído (OCR)</p>
		</div>
	{:else if filtered.length === 0}
		<div class="text-muted-foreground py-10 text-center flex flex-col items-center gap-3">
			<SearchIcon class="size-12 opacity-30" />
			<p>Sin resultados para "{query}"</p>
		</div>
	{:else if view === "grid"}
		<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each filtered as file (file.id)}
				<FileCard {file} href={`/document/${file.id}`} />
			{/each}
		</div>
	{:else}
		<div class="flex flex-col">
			{#each filtered as file (file.id)}
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
								<div class="bg-muted flex size-9 items-center justify-center rounded-2xl">
									<Icon class="text-muted-foreground size-4" />
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
