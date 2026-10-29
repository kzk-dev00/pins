// 表示を OFF にしたカテゴリの ID を localStorage に覚えておく。
// OFF の ID だけを持つので、ここにないカテゴリ（新しく作ったものも含む）は ON として扱う。

import { loadJSON, saveJSON } from "./storage";

const STORAGE_KEY = "pin-memo-hidden-categories";

function isArray(value: unknown): value is unknown[] {
  return Array.isArray(value);
}

export function loadHiddenCategoryIds(): string[] {
  const value = loadJSON(STORAGE_KEY, isArray) ?? [];
  return value.filter((id): id is string => typeof id === "string");
}

// 保存できなくても、次回はすべて ON で開くだけ
export function saveHiddenCategoryIds(ids: string[]): void {
  saveJSON(STORAGE_KEY, ids);
}
