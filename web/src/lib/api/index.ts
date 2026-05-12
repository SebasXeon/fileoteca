export { default as pb } from "$lib/pocketbase";
export { getRecentDocuments, getFavoriteDocuments, getSuggestedDocuments, searchDocuments, deleteDocument, getDocument, openDocument, getOpenUrl, toggleFavorite, updateDocumentCategory, updateDocumentNotes, getCategoriesWithSubcategories, type DocumentDetail } from "./documents";
export { getCategories, getIcons, createCategory, getDocumentsByCategory } from "./categories";
export { getFolders, createFolder, deleteFolder } from "./folders";