import pb from "$lib/pocketbase";
import type { ExplorerFolder } from "$lib/types";
import { toExplorerFile } from "./documents";
import type { ExplorerFile } from "$lib/types";

function toExplorerFolder(record: Record<string, unknown>): ExplorerFolder {
	return {
		id: record.id as string,
		name: record.name as string,
		description: (record.description as string) ?? undefined,
		parentId: (record.parent_id as string) ?? undefined,
		count: 0,
		updatedAt: new Date(record.updated as string),
	};
}

export async function getFolders(): Promise<ExplorerFolder[]> {
	const [folderRecords, junctionRecords] = await Promise.all([
		pb.collection("folders").getFullList({ sort: "name" }),
		pb.collection("document_folders").getFullList(),
	]);

	const countMap: Record<string, number> = {};
	for (const jr of junctionRecords) {
		const fid = (jr as unknown as Record<string, unknown>).folder_id as string;
		countMap[fid] = (countMap[fid] || 0) + 1;
	}

	return folderRecords.map((r) => {
		const raw = r as unknown as Record<string, unknown>;
		return {
			id: raw.id as string,
			name: raw.name as string,
			description: (raw.description as string) ?? undefined,
			parentId: (raw.parent_id as string) ?? undefined,
			count: countMap[raw.id as string] || 0,
			updatedAt: new Date(raw.updated as string),
		};
	});
}

export async function getFolder(id: string): Promise<ExplorerFolder> {
	const [record, junctionRecords] = await Promise.all([
		pb.collection("folders").getOne(id),
		pb.collection("document_folders").getFullList({ filter: `folder_id = "${id}"` }),
	]);
	const raw = record as unknown as Record<string, unknown>;
	return {
		id: raw.id as string,
		name: raw.name as string,
		description: (raw.description as string) ?? undefined,
		parentId: (raw.parent_id as string) ?? undefined,
		count: junctionRecords.length,
		updatedAt: new Date(raw.updated as string),
	};
}

export async function getSubfolders(parentId: string): Promise<ExplorerFolder[]> {
	const records = await pb.collection("folders").getFullList({
		filter: `parent_id = "${parentId}"`,
		sort: "name",
	});
	return records.map((r) => toExplorerFolder(r as unknown as Record<string, unknown>));
}

export async function getDocumentsInFolder(folderId: string): Promise<ExplorerFile[]> {
	const junctionRecords = await pb.collection("document_folders").getFullList({
		filter: `folder_id = "${folderId}"`,
	});
	const docIds = junctionRecords
		.map((r) => (r as unknown as Record<string, unknown>).document_id as string)
		.filter(Boolean);

	if (docIds.length === 0) return [];

	const filter = docIds.map((id) => `id = "${id}"`).join(" || ");
	const result = await pb.collection("documents").getList(1, docIds.length, {
		filter,
		sort: "-created",
		expand: "category_id,subcategory_id",
	});
	return (result.items as Array<unknown>).map((r) =>
		toExplorerFile(r as Record<string, unknown>),
	);
}

export async function createFolder(name: string, parentId?: string): Promise<ExplorerFolder> {
	const data: Record<string, unknown> = { name };
	if (parentId) data.parent_id = parentId;
	const record = await pb.collection("folders").create(data);
	return toExplorerFolder(record as unknown as Record<string, unknown>);
}

export async function updateFolder(
	id: string,
	data: { name?: string; description?: string; parent_id?: string | null },
): Promise<ExplorerFolder> {
	const record = await pb.collection("folders").update(id, data);
	return toExplorerFolder(record as unknown as Record<string, unknown>);
}

export async function deleteFolder(id: string): Promise<void> {
	// Unset parent_id on children so they don't become orphans
	const children = await pb.collection("folders").getFullList({
		filter: `parent_id = "${id}"`,
	});
	for (const child of children) {
		await pb.collection("folders").update(child.id, { parent_id: null });
	}
	await pb.collection("folders").delete(id);
}

export async function addDocumentToFolder(documentId: string, folderId: string): Promise<void> {
	await pb.collection("document_folders").create({
		document_id: documentId,
		folder_id: folderId,
	});
}

export async function removeDocumentFromFolder(documentId: string, folderId: string): Promise<void> {
	const records = await pb.collection("document_folders").getFullList({
		filter: `document_id = "${documentId}" && folder_id = "${folderId}"`,
	});
	if (records.length > 0) {
		await pb.collection("document_folders").delete(records[0].id);
	}
}

export async function getFoldersForDocument(
	documentId: string,
): Promise<{ id: string; name: string }[]> {
	const records = await pb.collection("document_folders").getFullList({
		filter: `document_id = "${documentId}"`,
		expand: "folder_id",
	});
	return records.map((r) => {
		const raw = r as unknown as Record<string, unknown>;
		const expand = raw.expand as Record<string, Record<string, unknown>> | undefined;
		const folder = expand?.folder_id;
		return {
			id: raw.folder_id as string,
			name: (folder?.name as string) || "Unknown",
		};
	});
}
