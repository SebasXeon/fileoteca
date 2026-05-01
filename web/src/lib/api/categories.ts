import pb from "$lib/pocketbase";
import type { ExplorerCategory, ExplorerIcon } from "$lib/types";

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