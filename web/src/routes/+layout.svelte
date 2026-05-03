<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import AppSidebar from "$lib/components/layout/app-sidebar.svelte";
	import * as Breadcrumb from "$lib/components/ui/breadcrumb/index.js";
	import * as Sidebar from "$lib/components/ui/sidebar/index.js";
	import { page } from "$app/stores";
	import { titleForPath } from "$lib/app-nav";
	import { onMount } from "svelte";
	import { Toaster } from "svelte-sonner";
	import { SIDEBAR_COOKIE_NAME } from "$lib/components/ui/sidebar/constants.js";

	let { children } = $props();

	const currentTitle = $derived.by(() => titleForPath($page.url.pathname));
	let sidebarOpen = $state(false);

	onMount(() => {
		const cookieValue = document.cookie
			.split("; ")
			.find((row) => row.startsWith(`${SIDEBAR_COOKIE_NAME}=`))
			?.split("=")[1];

		if (cookieValue === "true") sidebarOpen = true;
		if (cookieValue === "false") sidebarOpen = false;
	});
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<Sidebar.Provider bind:open={sidebarOpen}>
  <AppSidebar />
  <Sidebar.Inset class="overflow-hidden">
    <header class="bg-background sticky top-0 flex shrink-0 items-center gap-2 border-b p-4">
			<Sidebar.Trigger class="-ml-2" />
      <Breadcrumb.Root>
        <Breadcrumb.List>
          <Breadcrumb.Item class="hidden md:block">
            <Breadcrumb.Link href="/" data-sveltekit-preload-data="hover">Inicio</Breadcrumb.Link>
          </Breadcrumb.Item>
					{#if currentTitle !== "Inicio"}
						<Breadcrumb.Separator class="hidden md:block" />
						<Breadcrumb.Item class="hidden md:block">
							<Breadcrumb.Page>{currentTitle}</Breadcrumb.Page>
						</Breadcrumb.Item>
					{/if}
        </Breadcrumb.List>
      </Breadcrumb.Root>
    </header>
    <div class="flex flex-1 flex-col gap-4 p-4">
      {@render children()}
    </div>
  </Sidebar.Inset>
  <Toaster />
</Sidebar.Provider>
