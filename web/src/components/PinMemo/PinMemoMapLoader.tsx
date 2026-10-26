"use client";

import dynamic from "next/dynamic";
import styles from "./PinMemoMap.module.css";

// MapLibre はブラウザでしか動かないので、サーバーでは描画しない（ssr: false はクライアントコンポーネントでのみ使える）
const PinMemoMap = dynamic(() => import("./PinMemoMap"), {
  ssr: false,
  loading: () => <div className={styles.loading}>地図を読み込んでいます…</div>,
});

export default function PinMemoMapLoader() {
  return <PinMemoMap />;
}
