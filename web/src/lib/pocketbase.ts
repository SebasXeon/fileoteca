import PocketBase from "pocketbase";

const pb = new PocketBase(import.meta.env.VITE_POCKETBASE_URL ?? "http://127.0.0.1:8090");
pb.autoCancellation(false);

// Evitar cache del navegador/proxy en peticiones GET
pb.beforeSend = function (url, options) {
	if (options.method?.toUpperCase() === "GET") {
		const separator = url.includes("?") ? "&" : "?";
		url = `${url}${separator}_nc=${Date.now()}`;
	}
	return { url, options };
};

export default pb;