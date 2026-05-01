import pb from "$lib/pocketbase";
import type { ExplorerFile, FileKind } from "$lib/types";

function toExplorerFile(record: Record<string, unknown>): ExplorerFile {
	const expand = record.expand as Record<string, Record<string, unknown>> | undefined;
	const categoryRecord = expand?.category_id as Record<string, unknown> | undefined;
	const subcategoryRecord = expand?.subcategory_id as Record<string, unknown> | undefined;
	const categoryName = (categoryRecord?.name as string) ?? "";
	const subName = (subcategoryRecord?.name as string) ?? "";
	const location = [categoryName, subName].filter(Boolean).join(" / ") || "Sin categoría";

	return {
		id: record.id as string,
		name: record.name as string,
		ext: (record.file_ext ?? "txt") as FileKind,
		sizeBytes: (record.file_size as number) ?? 0,
		updatedAt: new Date(record.updated as string),
		locationLabel: location,
		category: categoryName || undefined,
		favorite: Boolean(record.is_favorite),
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

export async function deleteDocument(id: string): Promise<void> {
	await pb.collection("documents").delete(id);
}