const API_BASE = "";

type JsonRequestInit = Omit<RequestInit, "body" | "headers"> & {
  body?: BodyInit | Record<string, unknown> | unknown[] | null;
  headers?: HeadersInit;
};

function buildApiUrl(path: string): string {
  return `${API_BASE}${path}`;
}

function mergeHeaders(headers?: HeadersInit): Headers {
  return new Headers(headers);
}

export async function request(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = mergeHeaders(init.headers);
  return fetch(buildApiUrl(path), {
    ...init,
    headers,
    credentials: "include",
  });
}

export async function requestJson<T>(path: string, init: JsonRequestInit = {}): Promise<{
  response: Response;
  data: T;
}> {
  const headers = mergeHeaders(init.headers);
  let body = init.body as BodyInit | null | undefined;
  if (
    init.body !== undefined &&
    init.body !== null &&
    !(init.body instanceof FormData) &&
    !(init.body instanceof Blob) &&
    !(init.body instanceof URLSearchParams) &&
    !(init.body instanceof ArrayBuffer) &&
    !ArrayBuffer.isView(init.body) &&
    !(typeof ReadableStream !== "undefined" && init.body instanceof ReadableStream)
  ) {
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
  const data = (await response.json()) as T;
  return { response, data };
}

export { buildApiUrl };
