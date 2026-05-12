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
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import { Separator } from "$lib/components/ui/separator/index.js";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import {
		getDocument,
		openDocument,
		getOpenUrl,
		toggleFavorite,
		updateDocumentCategory,
		deleteDocument,
		getCategoriesWithSubcategories,
		type DocumentDetail,
	} from "$lib/api";
	import { formatBytes, formatDate } from "$lib/types";


	let doc = $state<DocumentDetail | null>(null);
	let loading = $state(true);
	let error = $state("");
	let opening = $state(false);
	let opened = $state(false);
	let openCooldown = $state(false);
	let textContent = $state("");
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
