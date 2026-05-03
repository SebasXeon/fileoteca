<script lang="ts">
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";
	import type { Snippet } from "svelte";
	import StarIcon from "@lucide/svelte/icons/star";
	import StarOffIcon from "@lucide/svelte/icons/star-off";
	import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
	import TrashIcon from "@lucide/svelte/icons/trash";
	import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
	import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
	import CopyIcon from "@lucide/svelte/icons/copy";
	import * as ContextMenu from "$lib/components/ui/context-menu/index.js";
	import {
		toggleFavorite,
		updateDocumentCategory,
		deleteDocument,
		getCategoriesWithSubcategories,
		openDocument,
	} from "$lib/api";

	let {
		docId,
		docName = "",
		isFavorite = false,
		isVisualizable = false,
		onchange,
		onFavoriteToggle,
		children,
	}: {
		docId: string;
		docName?: string;
		isFavorite?: boolean;
		isVisualizable?: boolean;
		onchange: () => void;
		onFavoriteToggle?: (isFavorite: boolean) => void;
		children: Snippet;
	} = $props();

	let categories = $state<Awaited<ReturnType<typeof getCategoriesWithSubcategories>>>([]);

	$effect(() => {
		getCategoriesWithSubcategories().then((cats) => { categories = cats; }).catch(() => {});
	});

	async function handleToggleFavorite() {
		try {
			const newVal = await toggleFavorite(docId, isFavorite);
			isFavorite = newVal;
			toast.success(newVal ? "Agregado a favoritos" : "Eliminado de favoritos");
			if (onFavoriteToggle) {
				onFavoriteToggle(newVal);
			} else {
				onchange();
			}
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleMoveToCategory(categoryId: string, subcategoryId: string) {
		try {
			await updateDocumentCategory(docId, categoryId, subcategoryId);
			toast.success("Documento movido");
			onchange();
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleDelete() {
		if (!confirm(`¿Eliminar "${docName}" permanentemente?`)) return;
		try {
			await deleteDocument(docId);
			toast.success("Documento eliminado");
			onchange();
		} catch (err) {
			toast.error(`Error al eliminar: ${err}`);
		}
	}

	async function handleOpenExternally() {
		try {
			await openDocument(docId);
			toast.info("Abriendo documento…");
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}

	async function handleCopyPath() {
		const base = (await import("$lib/pocketbase")).default.baseUrl.replace(/\/$/, "");
		const url = `${base}/api/documents/open/${docId}`;
		await navigator.clipboard.writeText(url);
		toast.success("Ruta copiada");
	}
</script>

<ContextMenu.Root>
	<ContextMenu.Trigger class="block">
		{@render children?.()}
	</ContextMenu.Trigger>
	<ContextMenu.Content class="min-w-52">
		{#if isVisualizable}
			<ContextMenu.Item onclick={() => goto(`/document/${docId}`)}>
				<FolderOpenIcon class="mr-2 size-4" />
				Abrir documento
			</ContextMenu.Item>
			<ContextMenu.Separator />
		{/if}
		<ContextMenu.Item onclick={handleToggleFavorite}>
			{#if isFavorite}
				<StarOffIcon class="mr-2 size-4" />
				Quitar de favoritos
			{:else}
				<StarIcon class="mr-2 size-4" />
				Agregar a favoritos
			{/if}
		</ContextMenu.Item>
		<ContextMenu.Item onclick={handleOpenExternally}>
			<ExternalLinkIcon class="mr-2 size-4" />
			Abrir externamente
		</ContextMenu.Item>
		{#if categories.length > 0}
			<ContextMenu.Separator />
			<ContextMenu.Sub>
				<ContextMenu.SubTrigger class="flex items-center gap-2">
					<FolderOpenIcon class="size-4" />
					Mover a categoría
					<ChevronRightIcon class="ml-auto size-4" />
				</ContextMenu.SubTrigger>
				<ContextMenu.SubContent class="min-w-48">
					{#each categories as cat (cat.id)}
						<ContextMenu.Sub>
							<ContextMenu.SubTrigger class="flex items-center gap-2">
								<span style="background-color: {cat.color}" class="size-2 rounded-full shrink-0"></span>
								{cat.name}
								<ChevronRightIcon class="ml-auto size-3" />
							</ContextMenu.SubTrigger>
							<ContextMenu.SubContent class="min-w-40">
								{#each cat.subcategories as sub (sub.id)}
									<ContextMenu.Item onclick={() => handleMoveToCategory(cat.id, sub.id)}>
										{sub.name}
									</ContextMenu.Item>
								{/each}
							</ContextMenu.SubContent>
						</ContextMenu.Sub>
					{/each}
				</ContextMenu.SubContent>
			</ContextMenu.Sub>
		{/if}
		<ContextMenu.Separator />
		<ContextMenu.Item onclick={handleCopyPath}>
			<CopyIcon class="mr-2 size-4" />
			Copiar ruta
		</ContextMenu.Item>
		<ContextMenu.Separator />
		<ContextMenu.Item class="text-destructive" onclick={handleDelete}>
			<TrashIcon class="mr-2 size-4" />
			Eliminar
		</ContextMenu.Item>
	</ContextMenu.Content>
</ContextMenu.Root>
