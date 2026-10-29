// 入力の文字数の上限。API（api/util.go）の定数と同じ値にする

export const CATEGORY_NAME_MAX_LENGTH = 30;
export const TITLE_MAX_LENGTH = 20;
export const MEMO_MAX_LENGTH = 200;

// API と同じく、前後の空白を除いた文字数（コードポイントで数える）
export function textLength(text: string): number {
  return [...text.trim()].length;
}

export function isValidCategoryName(name: string): boolean {
  const length = textLength(name);
  return length >= 1 && length <= CATEGORY_NAME_MAX_LENGTH;
}
