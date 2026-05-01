<script lang="ts">
	import SearchIcon from "@lucide/svelte/icons/search";
	import StarIcon from "@lucide/svelte/icons/star";
	import SparklesIcon from "@lucide/svelte/icons/sparkles";
	import ClockIcon from "@lucide/svelte/icons/clock";
	import LayoutGridIcon from "@lucide/svelte/icons/layout-grid";
	import ListIcon from "@lucide/svelte/icons/list";
	import SlidersHorizontalIcon from "@lucide/svelte/icons/sliders-horizontal";
	import ArrowUpDownIcon from "@lucide/svelte/icons/arrow-up-down";

	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { ButtonGroup } from "$lib/components/ui/button-group/index.js";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import { Item } from "$lib/components/ui/item/index.js";

	import FileCard from "$lib/components/explorer/file-card.svelte";

	import {
		FAVORITE_FILES,
		RECENT_FILES,
		SUGGESTED_FILES,
		fileIconFor,
		formatBytes,
		formatDate,
		type ExplorerFile,
	} from "$lib/mock/explorer";

	type ViewMode = "grid" | "list";
	type SortMode = "recent" | "name" | "size";
	type TypeFilter = "all" | "pdf" | "docx" | "xlsx" | "image";

	let query = $state("");
	let view: ViewMode = $state("grid");
	let sort: SortMode = $state("recent");
	let typeFilter: TypeFilter = $state("all");

	function matchesType(file: ExplorerFile) {
		if (typeFilter === "all") return true;
		if (typeFilter === "image") return file.ext === "png" || file.ext === "jpg";
		return file.ext === typeFilter;
	}

	function matchesQuery(file: ExplorerFile) {
		const q = query.trim().toLowerCase();
		if (!q) return true;
		return (
			file.name.toLowerCase().includes(q) ||
			(file.locationLabel ?? "").toLowerCase().includes(q) ||
			(file.category ?? "").toLowerCase().includes(q)
		);
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

	const filteredRecent = $derived.by(() =>
		sortFiles(RECENT_FILES.filter(matchesQuery).filter(matchesType))
	);
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="min-w-0">
			<h1 class="text-2xl font-semibold leading-tight">Inicio</h1>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<Button variant="outline">Importar</Button>
			<Button>Subir</Button>
		</div>
	</div>

	<div class="flex flex-col gap-3">
		<div class="flex flex-wrap items-center gap-2">
			<div class="relative min-w-[280px] flex-1">
				<SearchIcon
					class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2"
				/>
				<Input
					bind:value={query}
					placeholder="Buscar por nombre, categoría o ubicación…"
					class="pl-9"
				/>
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
			<span>{filteredRecent.length} resultado(s)</span>
			<Separator orientation="vertical" class="h-4" />
			{#if typeFilter !== "all"}
				<Badge variant="outline">Tipo: {typeFilter}</Badge>
			{/if}
			{#if query.trim()}
				<Badge variant="outline">Búsqueda: “{query.trim()}”</Badge>
			{/if}
		</div>
	</div>

	<section class="flex flex-col gap-3">
		<div class="flex items-center justify-between gap-3">
			<div class="flex items-center gap-2">
				<ClockIcon class="text-muted-foreground size-4" />
				<h2 class="text-lg font-semibold">Recientes</h2>
			</div>
			<Button variant="ghost" size="sm">Ver todo</Button>
		</div>

		{#if view === "grid"}
			<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
				{#each filteredRecent as file (file.id)}
					<FileCard {file} />
				{/each}
			</div>
		{:else}
			<div class="flex flex-col">
				{#each filteredRecent as file (file.id)}
					{@const Icon = fileIconFor(file.ext)}
					<Item variant="muted" size="sm" class="rounded-3xl">
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
				{/each}
			</div>
		{/if}
	</section>

	<section class="grid gap-6 lg:grid-cols-2">
		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-2">
				<StarIcon class="text-muted-foreground size-4" />
				<h2 class="text-lg font-semibold">Favoritos</h2>
			</div>
			<div class="flex gap-3 overflow-x-auto pb-2">
				{#each FAVORITE_FILES as file (file.id)}
					<div class="min-w-[260px]">
						<FileCard {file} variant="compact" />
					</div>
				{/each}
			</div>
		</div>

		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-2">
				<SparklesIcon class="text-muted-foreground size-4" />
				<h2 class="text-lg font-semibold">Sugeridos</h2>
			</div>
			<div class="flex flex-col gap-3">
				{#each SUGGESTED_FILES.slice(0, 4) as file (file.id)}
					<FileCard {file} variant="suggested" />
				{/each}
			</div>
		</div>
	</section>
</div>

