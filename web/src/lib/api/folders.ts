import pb from "$lib/pocketbase";
import type { ExplorerFolder } from "$lib/types";

function toExplorerFolder(record: Record<string, unknown>): ExplorerFolder {
	return {
		id: record.id as string,
		name: record.name as string,
		parentId: (record.parent_id as string) ?? undefined,
		count: 0,
		updatedAt: new Date(record.updated as string),
	};
}

export async function getFolders(): Promise<ExplorerFolder[]> {
	const records = await pb.collection("folders").getFullList({ sort: "name" });
	return records.map((r) => toExplorerFolder(r as unknown as Record<string, unknown>));
}

export async function createFolder(name: string, parentId?: string): Promise<ExplorerFolder> {
	const data: Record<string, unknown> = { name };
	if (parentId) data.parent_id = parentId;
	const record = await pb.collection("folders").create(data);
	return toExplorerFolder(record as unknown as Record<string, unknown>);
}

export async function deleteFolder(id: string): Promise<void> {
	await pb.collection("folders").delete(id);
}