import { getGuestToken } from "./guestToken";

// 形は api/model.go の categoryJSON / pinJSON に合わせる
export type Category = {
  categoryId: string;
  name: string;
  color: string;
  sortOrder: number;
  viewKey?: string;
  createdAt: string;
  updatedAt: string;
  expiresAt?: number;
};

export type Pin = {
  pinId: string;
  categoryId: string;
  lat: number;
  lng: number;
  memo: string;
  createdAt: string;
  updatedAt: string;
  expiresAt?: number;
};

// API のエラー。message は画面にそのまま出せる日本語。
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL;

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  if (!BASE_URL) {
    throw new ApiError(0, "config", "API の URL が設定されていません（NEXT_PUBLIC_API_BASE_URL）");
  }

  const headers: Record<string, string> = { "x-guest-token": getGuestToken() };
  if (body !== undefined) headers["content-type"] = "application/json";

  let res: Response;
  try {
    res = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "network", "通信できませんでした。電波の状態を確認して、もう一度お試しください");
  }

  if (res.status === 204) return undefined as T;

  const data = await res.json().catch(() => null);
  if (!res.ok) {
    const error = data?.error;
    throw new ApiError(res.status, error?.code ?? "unknown", error?.message ?? "エラーが発生しました");
  }
  return data as T;
}

export function errorMessage(err: unknown): string {
  return err instanceof ApiError ? err.message : "エラーが発生しました";
}

export async function listCategories(): Promise<Category[]> {
  const data = await request<{ categories: Category[] }>("GET", "/categories");
  return data.categories;
}

export function createCategory(input: { name: string; color?: string }): Promise<Category> {
  return request("POST", "/categories", input);
}

export function updateCategory(categoryId: string, input: { name?: string; color?: string }): Promise<Category> {
  return request("PATCH", `/categories/${categoryId}`, input);
}

export function deleteCategory(categoryId: string): Promise<void> {
  return request("DELETE", `/categories/${categoryId}`);
}

export async function listPins(categoryId?: string): Promise<Pin[]> {
  const query = categoryId ? `?categoryId=${encodeURIComponent(categoryId)}` : "";
  const data = await request<{ pins: Pin[] }>("GET", `/pins${query}`);
  return data.pins;
}

export function createPin(categoryId: string, input: { lat: number; lng: number; memo?: string }): Promise<Pin> {
  return request("POST", `/categories/${categoryId}/pins`, input);
}

export function updatePin(
  categoryId: string,
  pinId: string,
  input: { lat?: number; lng?: number; memo?: string },
): Promise<Pin> {
  return request("PATCH", `/categories/${categoryId}/pins/${pinId}`, input);
}

export function deletePin(categoryId: string, pinId: string): Promise<void> {
  return request("DELETE", `/categories/${categoryId}/pins/${pinId}`);
}
