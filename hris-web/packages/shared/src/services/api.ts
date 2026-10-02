import { useSession } from "../stores/session";
import type { Page, Pagination } from "../types";
interface Envelope<T> {
  data: T;
  message?: string;
  error_code?: string;
  meta?: { pagination?: Pagination };
}
async function request<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<Envelope<T>> {
  const token = useSession.getState().token;
  const response = await fetch(`/api/v1${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  });
  const data = await response
    .json()
    .catch(() => ({ message: "Server tidak dapat dihubungi" }));
  if (!response.ok) {
    if (response.status === 401 && path != "/auth/login")
      useSession.getState().clear();
    throw new Error(data.message || "Permintaan gagal");
  }
  return data as Envelope<T>;
}
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  return (await request<T>(path, method, body)).data;
}
// apiPage reads a paginated list (?page=N) together with its pagination meta.
export async function apiPage<T>(path: string): Promise<Page<T>> {
  const res = await request<T[]>(path);
  const items = res.data ?? [];
  return {
    items,
    pagination: res.meta?.pagination ?? {
      current_page: 1,
      per_page: items.length,
      total_records: items.length,
      total_pages: 1,
    },
  };
}
// query builds a query string, skipping empty values.
export function query(params: Record<string, string | number | undefined>) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params))
    if (v !== undefined && v !== "") q.set(k, String(v));
  const s = q.toString();
  return s ? `?${s}` : "";
}

// upload sends one file (multipart field "file") and returns its record.
export async function upload(
  file: File,
): Promise<{ id: number; name: string }> {
  const token = useSession.getState().token;
  const body = new FormData();
  body.append("file", file);
  const response = await fetch("/api/v1/files", {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body,
  });
  const data = await response
    .json()
    .catch(() => ({ message: "Server tidak dapat dihubungi" }));
  if (!response.ok) throw new Error(data.message || "Unggah gagal");
  return data.data;
}
// download fetches an authenticated file and saves it under its name.
export async function download(id: number, name: string) {
  const token = useSession.getState().token;
  const response = await fetch(`/api/v1/files/${id}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!response.ok) throw new Error("File tidak dapat diunduh");
  const url = URL.createObjectURL(await response.blob());
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}
