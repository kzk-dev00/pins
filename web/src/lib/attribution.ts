import type { MapLibreEvent } from "maplibre-gl";

// 地図の帰属表示（© OpenStreetMap など）を、最初から ⓘ だけにたたむ。押したときだけ文字を出す。
// MapLibre のコンパクト表示は、そのままだと開いた状態で始まり、地図を動かしたときに初めてたたまれるため、
// 地図の読み込みが終わったところでたたむ。
// 帰属表示は無料で使う条件なので、消さずに ⓘ を地図の上に残す（design.md の 8. FE のメモ）
export const COMPACT_ATTRIBUTION = { compact: true } as const;

export function collapseAttribution(e: MapLibreEvent) {
  const attrib = e.target.getContainer().querySelector(".maplibregl-ctrl-attrib");
  attrib?.classList.remove("maplibregl-compact-show");
  attrib?.removeAttribute("open");
}
