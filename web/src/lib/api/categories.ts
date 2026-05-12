import pb from "$lib/pocketbase";
import type { ExplorerCategory, ExplorerFile, ExplorerIcon, FileKind } from "$lib/types";

function toDocFile(record: Record<string, unknown>): ExplorerFile {
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

function toExplorerCategory(
	record: Record<string, unknown>,
	subcategories: Record<string, unknown>[],
	documentCount: number,
): ExplorerCategory {
	const expand = record.expand as Record<string, Record<string, unknown>> | undefined;
	const iconRecord = expand?.icon_id as Record<string, unknown> | undefined;

	return {
		id: record.id as string,
		name: record.name as string,
		color: (record.color as string) ?? "#71717a",
		iconName: (iconRecord?.name as string) ?? undefined,
		count: documentCount,
		subcategories: subcategories.map((sub) => ({
			id: sub.id as string,
			name: sub.name as string,
			count: 0,
		})),
	};
}

export async function getCategories(): Promise<ExplorerCategory[]> {
	const [categories, subcategories] = await Promise.all([
		pb.collection("categories").getFullList({ expand: "icon_id" }),
		pb.collection("subcategories").getFullList({ expand: "category_id" }),
	]);

	const rawCategories = categories.map((c) => c as unknown as Record<string, unknown>);
	const rawSubcategories = subcategories.map((s) => s as unknown as Record<string, unknown>);

	const documentCounts = new Map<string, number>();
	for (const doc of await pb.collection("documents").getFullList({ fields: "category_id" })) {
		const rawDoc = doc as unknown as Record<string, unknown>;
		const cid = rawDoc.category_id as string;
		if (cid) documentCounts.set(cid, (documentCounts.get(cid) ?? 0) + 1);
	}

	return rawCategories.map((cat) => {
		const subs = rawSubcategories.filter((sub) => sub.category_id === cat.id);
		return toExplorerCategory(cat, subs, documentCounts.get(cat.id as string) ?? 0);
	});
}

export async function getIcons(): Promise<ExplorerIcon[]> {
	const records = await pb.collection("icons").getFullList();
	return records.map((r) => {
		const raw = r as unknown as Record<string, unknown>;
		return {
			id: raw.id as string,
			name: raw.name as string,
			label: (raw.label as string) ?? undefined,
		};
	});
}

export async function createCategory(name: string, description: string, color: string, tags: string[]): Promise<ExplorerCategory> {
	const record = await pb.collection("categories").create({
		name,
		description,
		color,
		tags,
	});
	const raw = record as unknown as Record<string, unknown>;
	return {
		id: raw.id as string,
		name: raw.name as string,
		color: (raw.color as string) ?? "#71717a",
		count: 0,
		subcategories: [],
	};
}

export async function getDocumentsByCategory(categoryId: string): Promise<ExplorerFile[]> {
	const result = await pb.collection("documents").getList(1, 50, {
		filter: `category_id = "${categoryId}"`,
		sort: "-created",
		expand: "category_id,subcategory_id",
	});
	return result.items.map((r) => toDocFile(r as unknown as Record<string, unknown>));
}