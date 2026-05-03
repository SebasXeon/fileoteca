<script lang="ts">
	import StarIcon from "@lucide/svelte/icons/star";
	import StarOffIcon from "@lucide/svelte/icons/star-off";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import {
		fileIconFor,
		formatBytes,
		formatDate,
		type ExplorerFile,
	} from "$lib/types";
	import type { HTMLAttributes } from "svelte/elements";
	import DocumentContextMenu from "./document-context-menu.svelte";
	import { toggleFavorite } from "$lib/api";
	import { toast } from "svelte-sonner";

	type Variant = "default" | "compact" | "suggested";

	let {
		file,
		variant = "default",
		href,
		class: className,
		onchange,
		onFavoriteToggle,
		...restProps
	}: {
		file: ExplorerFile;
		variant?: Variant;
		href?: string;
		class?: string;
		onchange?: () => void;
		onFavoriteToggle?: (isFavorite: boolean) => void;
	} & HTMLAttributes<HTMLDivElement> = $props();

	const Icon = $derived.by(() => fileIconFor(file.ext));
	const visualizable = $derived(["pdf", "png", "jpg", "jpeg", "gif", "bmp", "svg", "webp", "tiff", "ico", "txt", "csv", "md", "json", "xml", "html", "htm", "rtf"].includes(file.ext));

	async function handleToggleFavorite() {
		try {
			const newVal = await toggleFavorite(file.id, file.favorite ?? false);
			file = { ...file, favorite: newVal };
			toast.success(newVal ? "Agregado a favoritos" : "Eliminado de favoritos");
			if (onFavoriteToggle) {
				onFavoriteToggle(newVal);
			} else {
				onchange?.();
			}
		} catch (err) {
			toast.error(`Error: ${err}`);
		}
	}
</script>

{#snippet cardContent()}
	<Card.Root class={["shadow-sm", "relative", className].filter(Boolean).join(" ")} {...restProps}>
	{#if variant === "suggested"}
		<Card.Content class="pt-0">
			<div class="flex gap-3">
				<div class="bg-muted mt-6 flex size-10 items-center justify-center rounded-2xl">
					<Icon class="text-muted-foreground size-5" />
				</div>
				<div class="min-w-0 flex-1 py-6">
					<div class="flex items-start justify-between gap-2">
						<div class="min-w-0">
							<div class="truncate font-medium">{file.name}</div>
							<div class="text-muted-foreground truncate text-xs">
								{file.locationLabel}
							</div>
						</div>
						<Badge variant="secondary">{file.ext.toUpperCase()}</Badge>
					</div>
					{#if file.suggestedReason}
						<p class="text-muted-foreground mt-2 text-xs">{file.suggestedReason}</p>
					{/if}
				</div>
			</div>
		</Card.Content>
	{:else}
		<Card.Header>
			<div class="flex gap-3 min-w-0">
				<div class="bg-muted flex size-10 items-center justify-center rounded-2xl" aria-hidden="true">
					<Icon class="text-muted-foreground size-5" />
				</div>
				<div class="min-w-0 flex-1">
					<Card.Title class="truncate">{file.name}</Card.Title>
					<Card.Description class="truncate">{file.locationLabel}</Card.Description>
				</div>
			</div>
		</Card.Header>
		<Card.Content class="pt-3">
			{#if variant === "compact"}
				<div class="text-muted-foreground flex items-center justify-between text-xs">
					<span>{formatDate(file.updatedAt)}</span>
					<Badge variant="secondary">{file.ext.toUpperCase()}</Badge>
				</div>
			{:else}
				<div class="text-muted-foreground flex flex-wrap items-center gap-2 text-xs">
					<Badge variant="secondary">{file.ext.toUpperCase()}</Badge>
					<span>{formatBytes(file.sizeBytes)}</span>
					<span>·</span>
					<span>{formatDate(file.updatedAt)}</span>
				</div>
			{/if}
		</Card.Content>
	{/if}

	<button
		onclick={(e) => { e.stopPropagation(); e.preventDefault(); handleToggleFavorite(); }}
		class="absolute bottom-2 right-2 opacity-0 group-hover/card:opacity-100 transition-opacity duration-200 flex items-center justify-center size-7 rounded-full bg-background/80 backdrop-blur-sm shadow-sm ring-1 ring-foreground/10 hover:bg-background"
		aria-label={file.favorite ? "Quitar de favoritos" : "Agregar a favoritos"}
	>
		{#if file.favorite}
			<StarIcon class="size-4 fill-amber-400 text-amber-400" />
		{:else}
			<StarOffIcon class="size-4 text-muted-foreground" />
		{/if}
	</button>
</Card.Root>
{/snippet}

<DocumentContextMenu
	docId={file.id}
	docName={file.name}
	isFavorite={file.favorite ?? false}
	isVisualizable={visualizable}
	onchange={() => onchange?.()}
	onFavoriteToggle={(val) => onFavoriteToggle?.(val)}
>
	{#snippet children()}
		{#if href}
			<a href={href} class="block no-underline text-inherit">
				{@render cardContent()}
			</a>
		{:else}
			{@render cardContent()}
		{/if}
	{/snippet}
</DocumentContextMenu>
