import FileArchiveIcon from "@lucide/svelte/icons/file-archive";
import FileImageIcon from "@lucide/svelte/icons/file-image";
import FileSpreadsheetIcon from "@lucide/svelte/icons/file-spreadsheet";
import FileTextIcon from "@lucide/svelte/icons/file-text";
import FileIcon from "@lucide/svelte/icons/file";
import FolderIcon from "@lucide/svelte/icons/folder";

export type FileKind = "pdf" | "docx" | "xlsx" | "png" | "jpg" | "zip" | "txt";

export type ExplorerFile = {
	id: string;
	name: string;
	ext: FileKind;
	sizeBytes: number;
	updatedAt: Date;
	locationLabel: string;
	category?: string;
	favorite?: boolean;
	suggestedReason?: string;
};

export type ExplorerCategory = {
	id: string;
	name: string;
	color: string; // tailwind class (bg-*)
	count: number;
	subcategories: { name: string; count: number }[];
};

export type ExplorerFolder = {
	id: string;
	name: string;
	parentId?: string;
	count: number;
	updatedAt: Date;
};

export const CATEGORIES: ExplorerCategory[] = [
	{
		id: "cat_inbox",
		name: "Inbox",
		color: "bg-zinc-950 dark:bg-zinc-50",
		count: 34,
		subcategories: [
			{ name: "default", count: 18 },
			{ name: "por_clasificar", count: 16 },
		],
	},
	{
		id: "cat_facturas",
		name: "Facturas",
		color: "bg-emerald-600",
		count: 92,
		subcategories: [
			{ name: "servicios", count: 30 },
			{ name: "compras", count: 62 },
		],
	},
	{
		id: "cat_contratos",
		name: "Contratos",
		color: "bg-indigo-600",
		count: 27,
		subcategories: [
			{ name: "laboral", count: 10 },
			{ name: "arriendo", count: 9 },
			{ name: "otros", count: 8 },
		],
	},
	{
		id: "cat_identidad",
		name: "Identidad",
		color: "bg-amber-500",
		count: 14,
		subcategories: [
			{ name: "personal", count: 8 },
			{ name: "familia", count: 6 },
		],
	},
	{
		id: "cat_proyectos",
		name: "Proyectos",
		color: "bg-sky-600",
		count: 48,
		subcategories: [
			{ name: "briefs", count: 12 },
			{ name: "entregables", count: 36 },
		],
	},
];

export const RECENT_FILES: ExplorerFile[] = [
	{
		id: "f_001",
		name: "Contrato de arriendo - Apt 1203",
		ext: "pdf",
		sizeBytes: 2_184_224,
		updatedAt: new Date(2026, 3, 14, 14, 35),
		locationLabel: "Contratos / arriendo",
		category: "Contratos",
		favorite: true,
	},
	{
		id: "f_002",
		name: "Factura Internet Abril 2026",
		ext: "pdf",
		sizeBytes: 884_736,
		updatedAt: new Date(2026, 3, 14, 9, 10),
		locationLabel: "Facturas / servicios",
		category: "Facturas",
		favorite: true,
	},
	{
		id: "f_003",
		name: "Inventario hogar",
		ext: "xlsx",
		sizeBytes: 327_680,
		updatedAt: new Date(2026, 3, 13, 20, 2),
		locationLabel: "Inbox",
		category: "Inbox",
	},
	{
		id: "f_004",
		name: "Cédula (escaneo)",
		ext: "jpg",
		sizeBytes: 1_572_864,
		updatedAt: new Date(2026, 3, 12, 12, 40),
		locationLabel: "Identidad / personal",
		category: "Identidad",
	},
	{
		id: "f_005",
		name: "Entrega - propuesta v3",
		ext: "docx",
		sizeBytes: 610_304,
		updatedAt: new Date(2026, 3, 11, 18, 22),
		locationLabel: "Proyectos / entregables",
		category: "Proyectos",
		suggestedReason: "Detecté palabras clave: “propuesta”, “entrega”, “v3”.",
	},
	{
		id: "f_006",
		name: "Soportes - impuestos 2025",
		ext: "zip",
		sizeBytes: 24_975_872,
		updatedAt: new Date(2026, 3, 10, 8, 5),
		locationLabel: "Inbox",
		category: "Inbox",
		suggestedReason: "Archivo comprimido con recibos y PDFs; suena a carpeta de soporte.",
	},
];

export const FAVORITE_FILES = RECENT_FILES.filter((f) => f.favorite);

export const SUGGESTED_FILES: ExplorerFile[] = [
	...RECENT_FILES.filter((f) => f.suggestedReason),
	{
		id: "f_007",
		name: "Factura energía Marzo 2026",
		ext: "pdf",
		sizeBytes: 792_576,
		updatedAt: new Date(2026, 3, 9, 7, 30),
		locationLabel: "Inbox",
		category: "Inbox",
		suggestedReason: "Parece factura (patrón de nombre + PDF).",
	},
];

export const FOLDERS: ExplorerFolder[] = [
	{
		id: "fold_01",
		name: "Personal",
		count: 34,
		updatedAt: new Date(2026, 3, 14, 13, 10),
	},
	{
		id: "fold_02",
		name: "Trabajo",
		count: 58,
		updatedAt: new Date(2026, 3, 13, 16, 45),
	},
	{
		id: "fold_03",
		name: "Impuestos",
		parentId: "fold_01",
		count: 12,
		updatedAt: new Date(2026, 3, 10, 9, 2),
	},
	{
		id: "fold_04",
		name: "Proyectos",
		parentId: "fold_02",
		count: 21,
		updatedAt: new Date(2026, 3, 11, 20, 5),
	},
];

export function formatBytes(bytes: number): string {
	if (!Number.isFinite(bytes)) return "—";
	const units = ["B", "KB", "MB", "GB", "TB"];
	let value = bytes;
	let i = 0;
	while (value >= 1024 && i < units.length - 1) {
		value /= 1024;
		i++;
	}
	const decimals = i === 0 ? 0 : value < 10 ? 1 : 0;
	return `${value.toFixed(decimals)} ${units[i]}`;
}

export function formatDate(dt: Date): string {
	if (!(dt instanceof Date) || Number.isNaN(dt.getTime())) return "—";
	try {
		return new Intl.DateTimeFormat("es-CO", {
			day: "2-digit",
			month: "short",
			year: "numeric",
		}).format(dt);
	} catch {
		return dt.toISOString?.().slice(0, 10) ?? "—";
	}
}

export function fileIconFor(ext: FileKind) {
	switch (ext) {
		case "pdf":
		case "docx":
		case "txt":
			return FileTextIcon;
		case "xlsx":
			return FileSpreadsheetIcon;
		case "png":
		case "jpg":
			return FileImageIcon;
		case "zip":
			return FileArchiveIcon;
		default:
			return FileIcon;
	}
}

export { FolderIcon };
