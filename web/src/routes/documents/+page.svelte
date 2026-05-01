<script lang="ts">
	import SearchIcon from "@lucide/svelte/icons/search";
	import FolderKanbanIcon from "@lucide/svelte/icons/folder-kanban";
	import TagIcon from "@lucide/svelte/icons/tag";
	import ArrowUpDownIcon from "@lucide/svelte/icons/arrow-up-down";
	import FilterIcon from "@lucide/svelte/icons/filter";

	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";

	import { CATEGORIES, type ExplorerCategory } from "$lib/mock/explorer";

	type SortMode = "relevance" | "name" | "count";
	type ScopeFilter = "all" | "mvp" | "optional";

	let query = $state("");
	let selectedCategoryId: string | "all" = $state("all");
	let sort: SortMode = $state("relevance");
	let scope: ScopeFilter = $state("all");

	const visibleCategories = $derived.by(() => {
		const q = query.trim().toLowerCase();
		let cats = CATEGORIES.filter(
			(c) =>
				!q ||
				c.name.toLowerCase().includes(q) ||
				c.subcategories.some((s) => s.name.toLowerCase().includes(q))
		);

		// placeholder: "scope" no hace nada todavía, pero deja el UI listo
		if (scope !== "all") cats = cats;

		cats = [...cats];
		cats.sort((a, b) => {
			if (sort === "name") return a.name.localeCompare(b.name, "es");
			if (sort === "count") return b.count - a.count;
			return b.count - a.count;
		});
		return cats;
	});

	const selectedCategory: ExplorerCategory | null = $derived.by(() => {
		if (selectedCategoryId === "all") return null;
		return CATEGORIES.find((c) => c.id === selectedCategoryId) ?? null;
	});
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="min-w-0">
			<h1 class="text-2xl font-semibold leading-tight">Documentos</h1>
			<p class="text-muted-foreground">
				Explora por categorías (estilo nube) y luego bajamos a subcategorías.
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<Button variant="outline">Nueva categoría</Button>
			<Button>Subir</Button>
		</div>
	</div>

	<div class="grid gap-4 lg:grid-cols-[320px_1fr]">
		<Card.Root class="shadow-sm">
			<Card.Header class="pb-2">
				<Card.Title class="flex items-center gap-2">
					<FolderKanbanIcon class="text-muted-foreground size-4" />
					Categorías
				</Card.Title>
				<Card.Description>Filtra y navega rápido.</Card.Description>
			</Card.Header>
			<Card.Content class="pt-0">
				<div class="flex flex-col gap-3">
					<div class="relative">
						<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
						<Input bind:value={query} placeholder="Buscar categorías…" class="pl-9" />
					</div>

					<div class="flex items-center gap-2">
						<Button
							variant={selectedCategoryId === "all" ? "default" : "outline"}
							size="sm"
							onclick={() => (selectedCategoryId = "all")}
						>
							Todas
						</Button>
						<Button variant="outline" size="sm" onclick={() => (query = "")}>Limpiar</Button>
					</div>

					<Separator />

					<div class="flex flex-col gap-1">
						{#each visibleCategories as cat (cat.id)}
							<button
								type="button"
								class="hover:bg-muted/50 focus-visible:ring-ring/30 flex w-full items-center justify-between gap-3 rounded-3xl px-3 py-2 text-left text-sm outline-none focus-visible:ring-3"
								data-active={selectedCategoryId === cat.id}
								onclick={() => (selectedCategoryId = cat.id)}
							>
								<div class="flex min-w-0 items-center gap-2">
									<span class="{cat.color} size-2.5 shrink-0 rounded-full"></span>
									<span class="truncate font-medium">{cat.name}</span>
								</div>
								<Badge variant="secondary">{cat.count}</Badge>
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
					<Input placeholder="Buscar documentos (próximamente)..." class="pl-9" />
				</div>

				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button variant="outline" {...props}>
								<FilterIcon class="mr-2 size-4" />
								Alcance
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="end" class="min-w-56">
						<DropdownMenu.Label>Alcance</DropdownMenu.Label>
						<DropdownMenu.Separator />
						<DropdownMenu.RadioGroup bind:value={scope}>
							<DropdownMenu.RadioItem value="all">Todo</DropdownMenu.RadioItem>
							<DropdownMenu.RadioItem value="mvp">MVP</DropdownMenu.RadioItem>
							<DropdownMenu.RadioItem value="optional">Opcional</DropdownMenu.RadioItem>
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
							<DropdownMenu.RadioItem value="relevance">Relevancia</DropdownMenu.RadioItem>
							<DropdownMenu.RadioItem value="name">Nombre</DropdownMenu.RadioItem>
							<DropdownMenu.RadioItem value="count">Cantidad</DropdownMenu.RadioItem>
						</DropdownMenu.RadioGroup>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</div>

			<div class="flex flex-wrap items-center gap-2">
				<Badge variant="secondary">
					{selectedCategory ? selectedCategory.name : "Todas las categorías"}
				</Badge>
				<Badge variant="outline" class="inline-flex items-center gap-1">
					<TagIcon class="size-3.5" />
					Filtros listos (sin lógica aún)
				</Badge>
			</div>

			<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
				{#each (selectedCategory ? [selectedCategory] : visibleCategories) as cat (cat.id)}
					<Card.Root class="shadow-sm">
						<Card.Header class="pb-2">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0">
									<Card.Title class="flex items-center gap-2">
										<span class="{cat.color} size-2.5 rounded-full"></span>
										<span class="truncate">{cat.name}</span>
									</Card.Title>
									<Card.Description class="truncate">
										{cat.count} documento(s)
									</Card.Description>
								</div>
								<Button variant="ghost" size="sm">Abrir</Button>
							</div>
						</Card.Header>
						<Card.Content class="pt-0">
							<div class="flex flex-wrap gap-2">
								{#each cat.subcategories as sub (sub.name)}
									<Badge variant="secondary">{sub.name} · {sub.count}</Badge>
								{/each}
							</div>
						</Card.Content>
					</Card.Root>
				{/each}
			</div>
		</div>
	</div>
</div>
