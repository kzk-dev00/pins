import { readStorage, writeStorage } from "./storage";

const STORAGE_KEY = "pin-memo-guest-token";
const TOKEN_PATTERN = /^[A-Za-z0-9_-]{32,128}$/;

let cachedToken: string | null = null;

// ゲストトークンを返す。初回はランダムに作って localStorage に保存する。
// localStorage が使えない（プライベートブラウズなど）ときは、このタブの間だけメモリで持つ。
export function getGuestToken(): string {
  if (cachedToken) return cachedToken;

  const saved = readStorage(STORAGE_KEY);
  if (saved && TOKEN_PATTERN.test(saved)) {
    cachedToken = saved;
    return saved;
  }

  // UUID 2つ分で 72 文字（英数字と "-"）
  const token = crypto.randomUUID() + crypto.randomUUID();
  writeStorage(STORAGE_KEY, token); // 保存できなくても、このタブの間は使える
  cachedToken = token;
  return token;
}
