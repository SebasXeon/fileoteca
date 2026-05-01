<script lang="ts">
	import StarIcon from "@lucide/svelte/icons/star";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import {
		fileIconFor,
		formatBytes,
		formatDate,
		type ExplorerFile,
	} from "$lib/types";
	import type { HTMLAttributes } from "svelte/elements";

	type Variant = "default" | "compact" | "suggested";

	let {
		file,
		variant = "default",
		class: className,
		...restProps
	}: {
		file: ExplorerFile;
		variant?: Variant;
		class?: string;
	} & HTMLAttributes<HTMLDivElement> = $props();

	const Icon = $derived.by(() => fileIconFor(file.ext));
</script>

<Card.Root class={["shadow-sm", className].filter(Boolean).join(" ")} {...restProps}>
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
		{#if file.favorite && variant === "default"}
			<Card.Footer>
				<span class="text-muted-foreground inline-flex items-center gap-1 text-xs">
					<StarIcon class="size-3" />
					Favorito
				</span>
			</Card.Footer>
		{/if}
	{/if}
</Card.Root>
