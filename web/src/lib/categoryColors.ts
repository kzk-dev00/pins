// カテゴリの色。新しいカテゴリには、今あるカテゴリが使っていない色を、この順番で割り当てる
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

// 今あるカテゴリの色から、次に使う色を返す。
// 使われている数が一番少ない色のうち、この並びで最初のものを選ぶ。
// 8色より少なければ必ず使われていない色になり、8色を超えても（会員）なるべく重ならない
export function nextCategoryColor(usedColors: readonly string[]): string {
  const counts = CATEGORY_COLORS.map(
    (color) => usedColors.filter((used) => used.toUpperCase() === color).length,
  );
  const min = Math.min(...counts);
  return CATEGORY_COLORS[counts.indexOf(min)];
}
