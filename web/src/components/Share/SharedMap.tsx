"use client";

import { useEffect, useRef, useState, type CSSProperties } from "react";
import Map, { GeolocateControl, Marker, NavigationControl } from "react-map-gl/maplibre";
import type { LngLatBoundsLike } from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import { collapseAttribution } from "@/lib/attribution";
import { DEFAULT_VIEW } from "@/lib/config";
import { BASE_MAP_PROPS } from "@/lib/map";
import type { SharedCategory, SharedPin } from "@/lib/sharedCategory";
import { useEscapeKey } from "../PinMemo/useEscapeKey";
import { usePinFront } from "../PinMemo/usePinFront";
import { useSheetHeight } from "../PinMemo/useSheetHeight";
import PinMemoPinIcon from "../PinMemo/PinMemoPinIcon";
import mapStyles from "../PinMemo/PinMemoMap.module.css";
import SharedMenu from "./SharedMenu";
import styles from "./Shared.module.css";

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
  // 上はカテゴリ名、右は地図のボタンがあるのでその分の余白を多めに取る
  return { bounds, fitBoundsOptions: { padding: { top: 90, bottom: 40, left: 40, right: 70 }, maxZoom: 16 } };
}

type Props = { category: SharedCategory; pins: SharedPin[] };

// 共有ページの地図。見るだけ（ピンを立てる・編集・削除・移動はできない）
export default function SharedMap({ category, pins }: Props) {
  const [initialViewState] = useState(() => initialView(pins));
  const [openPin, setOpenPin] = useState<SharedPin | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  // ピンが重なったとき最後に触ったピンを一番上に出す
  const { bringToFront, zIndexOf } = usePinFront();
  const sheetOpen = openPin !== null;
  const [sheetRef, sheetHeight] = useSheetHeight<HTMLDivElement>(sheetOpen);
  // 見出しが見切れているか（見切れたら末尾に「･･･」を出す）。画面の幅が変わったら測り直す
  const titleRef = useRef<HTMLHeadingElement>(null);
  const [titleClipped, setTitleClipped] = useState(false);

  useEffect(() => {
    const el = titleRef.current;
    if (!el) return;
    const observer = new ResizeObserver(() => setTitleClipped(el.scrollWidth > el.clientWidth));
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  // Escでシートを閉じる（メニューを開いているときはメニュー側で閉じる）
  useEscapeKey(() => setOpenPin(null), sheetOpen && !menuOpen);

  return (
    <div
      className={mapStyles.root}
      data-sheet-open={sheetOpen || undefined}
      style={{ "--sheet-height": `${sheetHeight}px` } as CSSProperties}
    >
      <Map
        {...BASE_MAP_PROPS}
        initialViewState={initialViewState}
        onLoad={collapseAttribution} // 帰属表示は ⓘ だけにたたんでおき、押したときだけ出す
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
            style={{ zIndex: zIndexOf(pin.pinId) }}
            onClick={(e) => {
              e.originalEvent.stopPropagation(); // 地図のonClick（シートを閉じる）を発火させない
              bringToFront(pin.pinId);
              setOpenPin(pin);
            }}
          >
            <PinMemoPinIcon color={category.color} title={pin.title} tooltip={pin.memo || undefined} />
          </Marker>
        ))}
      </Map>

      <button
        type="button"
        className={mapStyles.menuButton}
        onClick={() => {
          setOpenPin(null);
          setMenuOpen(true);
        }}
        aria-label="メニューを開く"
      >
        <span aria-hidden>☰</span>
      </button>

      {menuOpen && <SharedMenu onClose={() => setMenuOpen(false)} />}

      <header className={styles.header}>
        <span className={styles.dot} style={{ background: category.color }} aria-hidden />
        <h1 ref={titleRef} className={styles.title} data-clipped={titleClipped || undefined}>
          共有：{category.name}
        </h1>
        <span className={styles.count}>{pins.length}件</span>
      </header>

      {openPin && (
        <div ref={sheetRef} className={mapStyles.sheet} role="dialog" aria-labelledby="shared-memo-title">
          <div className={mapStyles.sheetHeader}>
            <h2 id="shared-memo-title" className={mapStyles.sheetTitle}>
              {openPin.title || "メモ"}
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
