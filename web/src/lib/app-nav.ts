import FileIcon from "@lucide/svelte/icons/file";
import HouseIcon from "@lucide/svelte/icons/house";
import FoldersIcon from "@lucide/svelte/icons/folder";
import SearchIcon from "@lucide/svelte/icons/search";
import BoltIcon from "@lucide/svelte/icons/bolt";

export type AppNavItem = {
	title: string;
	href: string;
	icon: any;
};

export const APP_NAV: AppNavItem[] = [
	{ title: "Inicio", href: "/", icon: HouseIcon },
	{ title: "Documentos", href: "/documents", icon: FileIcon },
	{ title: "Carpetas", href: "/folders", icon: FoldersIcon },
	{ title: "Buscar", href: "/search", icon: SearchIcon },
	{ title: "Configuración", href: "/settings", icon: BoltIcon },
];

export function normalizePath(pathname: string): string {
	if (!pathname) return "/";
	if (pathname === "/") return "/";
	return pathname.endsWith("/") ? pathname.slice(0, -1) : pathname;
}

export function isActivePath(currentPathname: string, href: string): boolean {
	return normalizePath(currentPathname) === normalizePath(href);
}

export function titleForPath(pathname: string): string {
	const normalized = normalizePath(pathname);
	if (normalized.startsWith("/document/")) return "Documento";
	const match = APP_NAV.find((i) => normalizePath(i.href) === normalized);
	if (match) return match.title;
	if (normalized === "/") return "Inicio";
	return (
		decodeURIComponent(normalized.split("/").filter(Boolean).at(-1) ?? "Inicio")
			.replace(/[-_]+/g, " ")
			.replace(/\b\w/g, (c) => c.toUpperCase()) || "Inicio"
	);
}

