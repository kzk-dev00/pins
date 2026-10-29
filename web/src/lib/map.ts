// 自分の地図と共有ページで共通の、地図の設定。
// MapLibre はブラウザでしか動かないので、ssr: false で読み込むコンポーネントからだけ使う

import { setWorkerUrl } from "maplibre-gl";
import { COMPACT_ATTRIBUTION } from "./attribution";

// Worker は public/ から配信する（scripts/copy-maplibre-worker.mjs がコピーしている）
setWorkerUrl("/maplibre-gl-worker.mjs");

const MAP_STYLE = "https://tiles.openfreemap.org/styles/liberty";

// <Map> に共通で渡す設定
export const BASE_MAP_PROPS = {
  mapStyle: MAP_STYLE,
  style: { width: "100%", height: "100%" },
  attributionControl: COMPACT_ATTRIBUTION,
};
