import { useSession } from "../stores/session";
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
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
  return data.data as T;
}
