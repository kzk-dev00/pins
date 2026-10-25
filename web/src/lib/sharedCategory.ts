// 共有ページ（/share/[viewKey]）のデータを API から取る。サーバー側（ページと OGP のメタデータ）で使う。
// 見る人はゲストトークンを持っていなくてよいので、トークンは付けない。

// 形は api/model.go の sharedCategoryJSON / sharedPinJSON に合わせる（ownerId や期限は含まれない）
export type SharedCategory = { name: string; color: string };
export type SharedPin = {
  pinId: string;
  lat: number;
  lng: number;
  memo: string;
  createdAt: string;
  updatedAt: string;
};
export type SharedData = { category: SharedCategory; pins: SharedPin[] };

export type SharedResult = { status: "ok"; data: SharedData } | { status: "not_found" } | { status: "error" };

const VIEW_KEY_PATTERN = /^[A-Za-z0-9]{22}$/;
// 共有キーを作った直後は GSI の反映が遅れて 404 になることがあるので、1回だけ待ってやり直す
const RETRY_DELAY_MS = 1000;

async function fetchOnce(viewKey: string): Promise<SharedResult> {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!base) return { status: "error" };
  try {
    const res = await fetch(`${base}/share/${viewKey}`, { cache: "no-store" });
    if (res.status === 404) return { status: "not_found" };
    if (!res.ok) return { status: "error" };
    return { status: "ok", data: (await res.json()) as SharedData };
  } catch {
    return { status: "error" };
  }
}

export async function fetchSharedCategory(viewKey: string): Promise<SharedResult> {
  if (!VIEW_KEY_PATTERN.test(viewKey)) return { status: "not_found" };
  const first = await fetchOnce(viewKey);
  if (first.status !== "not_found") return first;
  await new Promise((resolve) => setTimeout(resolve, RETRY_DELAY_MS));
  return fetchOnce(viewKey);
}
