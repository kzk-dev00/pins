// 表示を OFF にしたカテゴリの ID を localStorage に覚えておく。
// OFF の ID だけを持つので、ここにないカテゴリ（新しく作ったものも含む）は ON として扱う。

const STORAGE_KEY = "pin-memo-hidden-categories";

export function loadHiddenCategoryIds(): string[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "[]");
    return Array.isArray(value) ? value.filter((id): id is string => typeof id === "string") : [];
  } catch {
    return [];
  }
}

export function saveHiddenCategoryIds(ids: string[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
  } catch {
    // 保存できなくても、次回はすべて ON で開くだけなので無視する
  }
}
