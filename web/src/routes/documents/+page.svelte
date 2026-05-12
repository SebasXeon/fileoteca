<script lang="ts">
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import SearchIcon from "@lucide/svelte/icons/search";
	import FolderKanbanIcon from "@lucide/svelte/icons/folder-kanban";
	import PlusIcon from "@lucide/svelte/icons/plus";
	import XIcon from "@lucide/svelte/icons/x";

	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Label } from "$lib/components/ui/label/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";

	import { getCategories, createCategory } from "$lib/api";
	import type { ExplorerCategory } from "$lib/types";

	let query = $state("");
	let debouncedQuery = $state("");
	let selectedCategoryId: string | "all" = $state("all");
	let categories = $state<ExplorerCategory[]>([]);
	let loading = $state(true);

	let showCreateDialog = $state(false);
	let newName = $state("");
	let newDesc = $state("");
	let newColor = $state("#6366f1");
	let newTags = $state("");
	let creating = $state(false);

	const COLORS = [
		"#ef4444", "#f97316", "#f59e0b", "#84cc16", "#22c55e", "#06b6d4",
		"#3b82f6", "#6366f1", "#8b5cf6", "#d946ef", "#ec4899", "#6b7280",
	];

	$effect(() => {
		const val = query;
		const timer = setTimeout(() => { debouncedQuery = val; }, 250);
		return () => clearTimeout(timer);
	});

	onMount(async () => {
		await loadCategories();
	});

	async function loadCategories() {
		try {
			categories = await getCategories();
		} catch (err) {
			console.error("Failed to load categories:", err);
		} finally {
			loading = false;
		}
	}

	const visibleCategories = $derived.by(() => {
		const q = debouncedQuery.trim().toLowerCase();
		if (!q) return categories;
		return categories.filter((c) =>
			c.name.toLowerCase().includes(q) ||
			c.subcategories.some((s) => s.name.toLowerCase().includes(q))
		);
	});

	const selectedCategory = $derived.by(() => {
		if (selectedCategoryId === "all") return null;
		return categories.find((c) => c.id === selectedCategoryId) ?? null;
	});

	async function handleCreateCategory() {
		const name = newName.trim();
		if (!name) return;
		creating = true;
		try {
			const tags = newTags.split(",").map((t) => t.trim()).filter(Boolean);
			await createCategory(name, newDesc.trim(), newColor, tags);
			await loadCategories();
			resetForm();
			showCreateDialog = false;
		} catch (err) {
			console.error("Failed to create category:", err);
		} finally {
			creating = false;
		}
	}

	function resetForm() {
		newName = "";
		newDesc = "";
		newColor = "#6366f1";
		newTags = "";
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="min-w-0">
			<h1 class="text-2xl font-semibold leading-tight">Documentos</h1>
			<p class="text-muted-foreground">
				Explora por categorías y navega a sus documentos.
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<Button variant="outline" onclick={() => (showCreateDialog = true)}>
				<PlusIcon class="mr-2 size-4" />
				Nueva categoría
			</Button>
		</div>
	</div>

	{#if loading}
		<div class="text-muted-foreground py-10 text-center">Cargando…</div>
	{:else}
		<div class="grid gap-4 lg:grid-cols-[320px_1fr]">
			<Card.Root class="shadow-sm h-fit">
				<Card.Header class="pb-2">
					<Card.Title class="flex items-center gap-2">
						<FolderKanbanIcon class="text-muted-foreground size-4" />
						Categorías
					</Card.Title>
					<Card.Description>{categories.length} categorías</Card.Description>
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
										<span style="background-color: {cat.color}" class="size-2.5 shrink-0 rounded-full"></span>
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
						<Input placeholder="Buscar documentos por nombre... (próximamente)" class="pl-9" />
					</div>
				</div>

				<div class="flex flex-wrap items-center gap-2">
					<Badge variant="secondary">
						{selectedCategory ? selectedCategory.name : "Todas las categorías"}
					</Badge>
				</div>

				{#if selectedCategory}
					<div class="flex flex-col gap-4">
						<Card.Root class="shadow-sm">
							<Card.Header>
								<div class="flex items-start justify-between gap-3">
									<div class="min-w-0">
										<Card.Title class="flex items-center gap-2">
											<span style="background-color: {selectedCategory.color}" class="size-3 rounded-full inline-block"></span>
											{selectedCategory.name}
										</Card.Title>
										<Card.Description>{selectedCategory.count} documento(s)</Card.Description>
									</div>
									<Button onclick={() => goto(`/category/${selectedCategory.id}`)}>Abrir</Button>
								</div>
							</Card.Header>
							<Card.Content>
								<div class="flex flex-wrap gap-2">
									{#each selectedCategory.subcategories as sub (sub.id)}
										<Badge variant="secondary">{sub.name} · {sub.count}</Badge>
									{/each}
								</div>
							</Card.Content>
						</Card.Root>
					</div>
				{:else}
					<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
						{#each visibleCategories as cat (cat.id)}
							<Card.Root class="shadow-sm">
								<Card.Header class="pb-2">
									<div class="flex items-start justify-between gap-3">
										<div class="min-w-0">
											<Card.Title class="flex items-center gap-2">
												<span style="background-color: {cat.color}" class="size-2.5 rounded-full inline-block"></span>
												<span class="truncate">{cat.name}</span>
											</Card.Title>
											<Card.Description class="truncate">
												{cat.count} documento(s)
											</Card.Description>
										</div>
										<Button variant="ghost" size="sm" onclick={() => goto(`/category/${cat.id}`)}>Abrir</Button>
									</div>
								</Card.Header>
								<Card.Content class="pt-0">
									<div class="flex flex-wrap gap-2">
										{#each cat.subcategories as sub (sub.id)}
											<Badge variant="secondary">{sub.name} · {sub.count}</Badge>
										{/each}
									</div>
								</Card.Content>
							</Card.Root>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<Dialog.Root open={showCreateDialog} onOpenChange={(o) => (showCreateDialog = o)}>
	<Dialog.Content class="sm:max-w-[420px]">
		<Dialog.Header>
			<Dialog.Title>Nueva categoría</Dialog.Title>
			<Dialog.Description>Crea una categoría para organizar tus documentos.</Dialog.Description>
		</Dialog.Header>
		<div class="flex flex-col gap-4 py-4">
			<div class="flex flex-col gap-2">
				<Label for="cat-name">Nombre</Label>
				<Input id="cat-name" bind:value={newName} placeholder="Ej: Finanzas" />
			</div>
			<div class="flex flex-col gap-2">
				<Label for="cat-desc">Descripción</Label>
				<Input id="cat-desc" bind:value={newDesc} placeholder="Opcional" />
			</div>
			<div class="flex flex-col gap-2">
				<Label>Color</Label>
				<div class="flex flex-wrap gap-2">
					{#each COLORS as color}
						<button
							type="button"
							class="size-8 rounded-full border-2 transition-all"
							class:border-foreground={newColor === color}
							class:border-transparent={newColor !== color}
							style="background-color: {color}"
							onclick={() => (newColor = color)}
							aria-label="Color {color}"
						></button>
					{/each}
				</div>
			</div>
			<div class="flex flex-col gap-2">
				<Label for="cat-tags">Etiquetas</Label>
				<Input id="cat-tags" bind:value={newTags} placeholder="Separadas por coma: tag1, tag2" />
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showCreateDialog = false)}>Cancelar</Button>
			<Button onclick={handleCreateCategory} disabled={creating || !newName.trim()}>
				{creating ? "Creando…" : "Crear categoría"}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
