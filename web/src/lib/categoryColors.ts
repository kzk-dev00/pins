// カテゴリの色。新しいカテゴリには、作るたびにこの順番で割り当てる
export const CATEGORY_COLORS = [
  "#E0472F", // 赤
  "#2F80E0", // 青
  "#2FA36B", // 緑
  "#E0A82F", // 黄
  "#8E4FE0", // 紫
  "#E04F9B", // ピンク
  "#1FA5A5", // 青緑
  "#6B7280", // 灰
] as const;

// 今あるカテゴリの数から、次に使う色を返す
export function nextCategoryColor(categoryCount: number): string {
  return CATEGORY_COLORS[categoryCount % CATEGORY_COLORS.length];
}
