import pb from "$lib/pocketbase";
import type { ExplorerFile, FileKind } from "$lib/types";

function toExplorerFile(record: Record<string, unknown>): ExplorerFile {
	const expand = record.expand as Record<string, Record<string, unknown>> | undefined;
	const categoryRecord = expand?.category_id as Record<string, unknown> | undefined;
	const subcategoryRecord = expand?.subcategory_id as Record<string, unknown> | undefined;
	const categoryName = (categoryRecord?.name as string) ?? "";
	const subName = (subcategoryRecord?.name as string) ?? "";
	const location = [categoryName, subName].filter(Boolean).join(" / ") || "Sin categoría";
	const rawThumb = record.thumbnail;
	const thumbUrl = (rawThumb && typeof rawThumb === "string" && rawThumb.length > 0)
		? `${pb.baseUrl}/api/files/documents/${record.id}/${rawThumb}`
		: undefined;

	return {
		id: record.id as string,
		name: record.name as string,
		ext: (record.file_ext ?? "txt") as FileKind,
		sizeBytes: (record.file_size as number) ?? 0,
		updatedAt: new Date(record.updated as string),
		locationLabel: location,
		category: categoryName || undefined,
		favorite: Boolean(record.is_favorite),
		thumbnail: thumbUrl,
	};
}

function mapItems(items: Array<unknown>): ExplorerFile[] {
	return items.map((r) => toExplorerFile(r as Record<string, unknown>));
}

export async function getRecentDocuments(): Promise<ExplorerFile[]> {
	const result = await pb.collection("documents").getList(1, 50, {
		sort: "-created",
		expand: "category_id,subcategory_id",
	});
	return mapItems(result.items);
}

export async function getFavoriteDocuments(): Promise<ExplorerFile[]> {
	try {
		const result = await pb.collection("documents").getList(1, 50, {
			filter: "is_favorite = true",
			sort: "-created",
			expand: "category_id,subcategory_id",
		});
		return mapItems(result.items);
	} catch {
		return [];
	}
}

export type DocumentDetail = ExplorerFile & {
	thumbnail?: string;
	path?: string;
	file?: string;
	notes?: string;
	ocr_txt?: string;
	metadata?: any;
	source_type?: string;
	status?: string;
	last_access?: string;
	category_id?: string;
	subcategory_id?: string;
};

export async function getDocument(id: string): Promise<DocumentDetail> {
	const record = await pb.collection("documents").getOne(id, {
		expand: "category_id,subcategory_id",
	});
	const raw = record as unknown as Record<string, unknown>;
	const expand = raw.expand as Record<string, Record<string, unknown>> | undefined;
	const categoryRecord = expand?.category_id as Record<string, unknown> | undefined;
	const subcategoryRecord = expand?.subcategory_id as Record<string, unknown> | undefined;
	const categoryName = (categoryRecord?.name as string) ?? "";
	const subName = (subcategoryRecord?.name as string) ?? "";
	const location = [categoryName, subName].filter(Boolean).join(" / ") || "Sin categoría";

	const rawThumb = raw.thumbnail;
	const thumbUrl = (rawThumb && typeof rawThumb === "string" && rawThumb.length > 0)
		? `${pb.baseUrl}/api/files/documents/${raw.id}/${rawThumb}`
		: undefined;

	return {
		id: raw.id as string,
		name: raw.name as string,
		ext: (raw.file_ext ?? "txt") as FileKind,
		sizeBytes: (raw.file_size as number) ?? 0,
		updatedAt: new Date(raw.updated as string),
		locationLabel: location,
		category: categoryName || undefined,
		favorite: Boolean(raw.is_favorite),
		thumbnail: thumbUrl,
		path: (raw.path as string) || undefined,
		file: (raw.file as string) || undefined,
		notes: (raw.notes as string) || undefined,
		ocr_txt: (raw.ocr_txt as string) || undefined,
		metadata: raw.metadata,
		source_type: (raw.source_type as string) || undefined,
		status: (raw.status as string) || undefined,
		last_access: (raw.last_access as string) || undefined,
		category_id: (raw.category_id as string) || undefined,
		subcategory_id: (raw.subcategory_id as string) || undefined,
	};
}

export function getOpenUrl(id: string): string {
	const base = pb.baseUrl.replace(/\/$/, "");
	return `${base}/api/documents/open/${id}`;
}

export async function openDocument(id: string): Promise<{ action: string }> {
	const resp = await fetch(getOpenUrl(id));
	if (!resp.ok) {
		let message = `Error ${resp.status}`;
		try {
			const contentType = resp.headers.get("content-type") || "";
			if (contentType.includes("application/json")) {
				const err = await resp.json();
				message = err.error || err.message || message;
			} else {
				const text = await resp.text();
				if (text) message = text.slice(0, 200);
			}
		} catch {
			// keep default message
		}
		throw new Error(message);
	}
	return resp.json();
}

export async function toggleFavorite(id: string, current: boolean): Promise<boolean> {
	const record = await pb.collection("documents").update(id, { is_favorite: !current });
	const raw = record as unknown as Record<string, unknown>;
	return Boolean(raw.is_favorite);
}

export async function updateDocumentCategory(id: string, categoryId: string, subcategoryId: string): Promise<void> {
	await pb.collection("documents").update(id, {
		category_id: categoryId,
		subcategory_id: subcategoryId,
	});
}

export async function updateDocumentNotes(id: string, notes: string): Promise<void> {
	await pb.collection("documents").update(id, { notes });
}

export async function getCategoriesWithSubcategories(): Promise<{ id: string; name: string; color: string; subcategories: { id: string; name: string }[] }[]> {
	const [categories, subcategories] = await Promise.all([
		pb.collection("categories").getFullList(),
		pb.collection("subcategories").getFullList(),
	]);
	const rawCategories = categories.map((c) => c as unknown as Record<string, unknown>);
	const rawSubs = subcategories.map((s) => s as unknown as Record<string, unknown>);

	return rawCategories.map((cat) => ({
		id: cat.id as string,
		name: cat.name as string,
		color: (cat.color as string) ?? "#71717a",
		subcategories: rawSubs
			.filter((s) => s.category_id === cat.id)
			.map((s) => ({
				id: s.id as string,
				name: s.name as string,
			})),
	}));
}

export async function deleteDocument(id: string): Promise<void> {
	await pb.collection("documents").delete(id);
}