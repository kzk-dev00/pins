// localStorage の読み書き。プライベートブラウズなどで使えないときも例外を出さない

// 読めないときは null
export function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

// 保存できなくても無視する（呼び出し側は、次回に初期状態から始まるだけで困らないものに使う）
export function writeStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // 何もしない
  }
}

// JSON で保存した値を返す。ない・壊れている・isValid で弾かれたときは null
export function loadJSON<T>(key: string, isValid: (value: unknown) => value is T): T | null {
  const raw = readStorage(key);
  if (!raw) return null;
  try {
    const value: unknown = JSON.parse(raw);
    return isValid(value) ? value : null;
  } catch {
    return null;
  }
}

export function saveJSON(key: string, value: unknown): void {
  writeStorage(key, JSON.stringify(value));
}
