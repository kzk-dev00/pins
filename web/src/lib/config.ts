// 画面の設定。NEXT_PUBLIC_ の値はビルド時に埋め込まれるので、変えたら再ビルドが必要。
// ローカルは .env.local、本番は Amplify Hosting の環境変数に設定する。

// API Gateway の URL。未設定なら undefined（呼び出し側でエラーにする）
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL;

// 環境変数がないとき・おかしいときの初期位置（東京駅）
const FALLBACK_VIEW = { latitude: 35.6812, longitude: 139.7671, zoom: 14 };

// 数値で、範囲内のときだけ使う。それ以外は fallback にする
function numberInRange(value: string | undefined, min: number, max: number, fallback: number): number {
  if (value === undefined || value.trim() === "") return fallback;
  const n = Number(value);
  return Number.isFinite(n) && n >= min && n <= max ? n : fallback;
}

// process.env.NEXT_PUBLIC_xxx は、このように直接書かないとビルド時に埋め込まれない
export const DEFAULT_VIEW = {
  latitude: numberInRange(process.env.NEXT_PUBLIC_DEFAULT_LAT, -90, 90, FALLBACK_VIEW.latitude),
  longitude: numberInRange(process.env.NEXT_PUBLIC_DEFAULT_LNG, -180, 180, FALLBACK_VIEW.longitude),
  zoom: numberInRange(process.env.NEXT_PUBLIC_DEFAULT_ZOOM, 0, 22, FALLBACK_VIEW.zoom),
};
