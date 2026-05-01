<script lang="ts">
	import { page } from "$app/stores";
	import CommandIcon from "@lucide/svelte/icons/command";
	import type { ComponentProps } from "svelte";
	import { APP_NAV, isActivePath } from "$lib/app-nav";
	import { useSidebar } from "$lib/components/ui/sidebar/context.svelte.js";
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import NavUser from "./nav-user.svelte";

	let { ref = $bindable(null), ...restProps }: ComponentProps<
		typeof Sidebar.Root
	> = $props();

	const sidebar = useSidebar();
	const currentPath = $derived.by(() => $page.url.pathname);

	const user = {
		name: "Sebas",
		email: "sebas@example.com",
		avatar: "/avatars/user.jpg",
	};
</script>

<Sidebar.Root bind:ref collapsible="icon" variant="inset" {...restProps}>
	<Sidebar.Header>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Sidebar.MenuButton size="lg" class="md:h-8 md:p-0">
					{#snippet child({ props })}
						<a href="/" {...props} data-sveltekit-preload-data="hover">
							<div
								class="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg"
							>
								<CommandIcon class="size-4" />
							</div>
							<div class="grid flex-1 text-start text-sm leading-tight">
								<span class="truncate font-medium">Fileoteca</span>
								<span class="truncate text-xs">Local-first</span>
							</div>
						</a>
					{/snippet}
				</Sidebar.MenuButton>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Header>

	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.GroupContent class="px-1.5 md:px-0">
				<Sidebar.Menu>
					{#each APP_NAV as item (item.href)}
						{@const active = isActivePath(currentPath, item.href)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton
								isActive={active}
								tooltipContent={item.title}
								class="px-2.5 md:px-2"
							>
								{#snippet child({ props })}
									<a
										href={item.href}
										aria-current={active ? "page" : undefined}
										data-sveltekit-preload-data="hover"
										onclick={() => sidebar.isMobile && sidebar.setOpenMobile(false)}
										{...props}
									>
										<item.icon />
										<span>{item.title}</span>
									</a>
								{/snippet}
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					{/each}
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>
	</Sidebar.Content>

	<Sidebar.Footer>
		<NavUser user={user} />
	</Sidebar.Footer>
</Sidebar.Root>
