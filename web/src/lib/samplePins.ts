// 初めて開いたときに「サンプルカテゴリ」へ入れる見本のピン（design.md の 8. FE のメモ）。
// 位置は固定ではなく、最初の位置（現在地、取れなければ初期位置）からのずれ（m）で決める。

// タイトルは20文字まで、メモは200文字まで
export const SAMPLE_PINS = [
  {
    title: "地図をタップで追加",
    memo: "何もないところをタップするとピンを立ててメモを残せます。",
    north: 180,
    east: -220,
  },
  {
    title: "長押しで移動できます",
    memo: "ピンは長押ししてドラッグすると場所を直せます。",
    north: -200,
    east: -120,
  },
  {
    title: "☰ から共有",
    memo: "右上の ☰ から、カテゴリごとに他の人とURLで共有できます。（共有された相手はピンの編集ができません）",
    north: 60,
    east: 200,
  },
] as const;

const METERS_PER_DEGREE_LAT = 111_320;

// 基準の位置から、北へ north m・東へ east m ずらした緯度経度を返す
export function offsetPosition(lat: number, lng: number, north: number, east: number) {
  const dLat = north / METERS_PER_DEGREE_LAT;
  // 経度1度の長さは緯度によって変わる（赤道で最長、極で0）
  const dLng = east / (METERS_PER_DEGREE_LAT * Math.cos((lat * Math.PI) / 180));
  return { lat: lat + dLat, lng: lng + dLng };
}
