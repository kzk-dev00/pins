"use client";

import { useEffect, useState, type CSSProperties } from "react";
import Map, { GeolocateControl, Marker, NavigationControl } from "react-map-gl/maplibre";
import { setWorkerUrl, type LngLatBoundsLike } from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import { DEFAULT_VIEW } from "@/lib/config";
import type { SharedCategory, SharedPin } from "@/lib/sharedCategory";
import { useSheetHeight } from "../PinMemo/useSheetHeight";
import mapStyles from "../PinMemo/Map.module.css";
import styles from "./Shared.module.css";

// Worker は public/ から配信する（scripts/copy-maplibre-worker.mjs がコピーしている）
setWorkerUrl("/maplibre-gl-worker.mjs");

const MAP_STYLE = "https://tiles.openfreemap.org/styles/liberty";
const SINGLE_PIN_ZOOM = 15;

// 初期位置：2本以上なら全部が収まる範囲、1本ならそのピンを中心にズーム15、0本なら環境変数の初期位置。
// 「最後に見ていた位置」は使わない（自分の地図の位置を上書きしないため）
function initialView(pins: SharedPin[]) {
  if (pins.length === 0) return DEFAULT_VIEW;
  if (pins.length === 1) return { latitude: pins[0].lat, longitude: pins[0].lng, zoom: SINGLE_PIN_ZOOM };
  const lngs = pins.map((p) => p.lng);
  const lats = pins.map((p) => p.lat);
  const bounds: LngLatBoundsLike = [
    [Math.min(...lngs), Math.min(...lats)],
    [Math.max(...lngs), Math.max(...lats)],
  ];
  // 上はカテゴリ名、右は地図のボタンがあるので、その分の余白を多めに取る
  return { bounds, fitBoundsOptions: { padding: { top: 90, bottom: 40, left: 40, right: 70 }, maxZoom: 16 } };
}

type Props = { category: SharedCategory; pins: SharedPin[] };

// 共有ページの地図。見るだけ（ピンを立てる・編集・削除・移動はできない）
export default function SharedMap({ category, pins }: Props) {
  const [initialViewState] = useState(() => initialView(pins));
  const [openPin, setOpenPin] = useState<SharedPin | null>(null);
  const sheetOpen = openPin !== null;
  const [sheetRef, sheetHeight] = useSheetHeight<HTMLDivElement>(sheetOpen);

  // Esc でシートを閉じる
  useEffect(() => {
    if (!sheetOpen) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenPin(null);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [sheetOpen]);

  return (
    <div
      className={mapStyles.root}
      data-sheet-open={sheetOpen || undefined}
      style={{ "--sheet-height": `${sheetHeight}px` } as CSSProperties}
    >
      <Map
        initialViewState={initialViewState}
        mapStyle={MAP_STYLE}
        style={{ width: "100%", height: "100%" }}
        onClick={() => setOpenPin(null)} // シートの外（地図）をタップしたら閉じる。ピンは立てない
      >
        <NavigationControl position="top-right" />
        {/* 現在地へは自動で移動しない。ボタンを押したときだけ */}
        <GeolocateControl position="top-right" positionOptions={{ enableHighAccuracy: true }} showUserLocation />

        {pins.map((pin) => (
          <Marker
            key={pin.pinId}
            latitude={pin.lat}
            longitude={pin.lng}
            anchor="bottom-left"
            onClick={(e) => {
              e.originalEvent.stopPropagation(); // 地図の onClick（シートを閉じる）を発火させない
              setOpenPin(pin);
            }}
          >
            <div
              className={mapStyles.pin}
              style={{ "--pin-color": category.color } as CSSProperties}
              title={pin.memo || "(メモなし)"}
            />
          </Marker>
        ))}
      </Map>

      <header className={styles.header}>
        <span className={styles.dot} style={{ background: category.color }} aria-hidden />
        <h1 className={styles.title}>{category.name}</h1>
        <span className={styles.count}>{pins.length}件</span>
      </header>

      {openPin && (
        <div ref={sheetRef} className={mapStyles.sheet} role="dialog" aria-labelledby="shared-memo-title">
          <div className={mapStyles.sheetHeader}>
            <h2 id="shared-memo-title" className={mapStyles.sheetTitle}>
              メモ
            </h2>
            <button type="button" className={mapStyles.sheetClose} onClick={() => setOpenPin(null)} aria-label="閉じる">
              ×
            </button>
          </div>
          <p className={styles.memo}>{openPin.memo || "（メモなし）"}</p>
        </div>
      )}
    </div>
  );
}
