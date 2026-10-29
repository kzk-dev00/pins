// 最後に見ていた地図の位置を localStorage に覚えておき、次に開いたときにそこから表示する。

import { loadJSON, saveJSON } from "./storage";

const STORAGE_KEY = "pin-memo-last-view";

export type MapView = { latitude: number; longitude: number; zoom: number };

function isValidView(v: unknown): v is MapView {
  if (typeof v !== "object" || v === null) return false;
  const { latitude, longitude, zoom } = v as Record<string, unknown>;
  return (
    typeof latitude === "number" && latitude >= -90 && latitude <= 90 &&
    typeof longitude === "number" && longitude >= -180 && longitude <= 180 &&
    typeof zoom === "number" && zoom >= 0 && zoom <= 22
  );
}

// 保存した位置を返す。ない・壊れている・読めないときは null
export function loadLastView(): MapView | null {
  return loadJSON(STORAGE_KEY, isValidView);
}

// 保存できなくても、次回は初期位置から開くだけ
export function saveLastView(view: MapView): void {
  const { latitude, longitude, zoom } = view;
  saveJSON(STORAGE_KEY, { latitude, longitude, zoom });
}
