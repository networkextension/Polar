const API_BASE = "";
function buildApiUrl(path) {
    return `${API_BASE}${path}`;
}
function mergeHeaders(headers) {
    return new Headers(headers);
}
export async function request(path, init = {}) {
    const headers = mergeHeaders(init.headers);
    return fetch(buildApiUrl(path), {
        ...init,
        headers,
        credentials: "include",
    });
}
export async function requestJson(path, init = {}) {
    const headers = mergeHeaders(init.headers);
    let body = init.body;
    if (init.body !== undefined &&
        init.body !== null &&
        !(init.body instanceof FormData) &&
        !(init.body instanceof Blob) &&
        !(init.body instanceof URLSearchParams) &&
        !(init.body instanceof ArrayBuffer) &&
        !ArrayBuffer.isView(init.body) &&
        !(typeof ReadableStream !== "undefined" && init.body instanceof ReadableStream)) {
        body = JSON.stringify(init.body);
    }
    if (init.body !== undefined && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
        headers.set("Content-Type", "application/json");
    }
    const response = await request(path, {
        ...init,
        headers,
        body,
    });
    const data = (await response.json());
    return { response, data };
}
export { buildApiUrl };
