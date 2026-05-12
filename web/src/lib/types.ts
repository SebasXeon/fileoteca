import FileArchiveIcon from "@lucide/svelte/icons/file-archive";
import FileImageIcon from "@lucide/svelte/icons/file-image";
import FileSpreadsheetIcon from "@lucide/svelte/icons/file-spreadsheet";
import FileTextIcon from "@lucide/svelte/icons/file-text";
import FileIcon from "@lucide/svelte/icons/file";

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
	thumbnail?: string;
	suggestedReason?: string;
};

export type ExplorerCategory = {
	id: string;
	name: string;
	color: string;
	iconName?: string;
	count: number;
	subcategories: { id: string; name: string; count: number }[];
};

export type ExplorerFolder = {
	id: string;
	name: string;
	parentId?: string;
	count: number;
	updatedAt: Date;
};

export type ExplorerIcon = {
	id: string;
	name: string;
	label?: string;
};

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

export { default as FolderIcon } from "@lucide/svelte/icons/folder";