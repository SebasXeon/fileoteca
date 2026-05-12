export { default as pb } from "$lib/pocketbase";
export { getRecentDocuments, getFavoriteDocuments, getSuggestedDocuments, deleteDocument, getDocument, openDocument, getOpenUrl, toggleFavorite, updateDocumentCategory, updateDocumentNotes, getCategoriesWithSubcategories, type DocumentDetail } from "./documents";
export { getCategories, getIcons } from "./categories";
export { getFolders, createFolder, deleteFolder } from "./folders";