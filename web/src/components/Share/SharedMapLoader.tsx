"use client";

import dynamic from "next/dynamic";
import type { SharedCategory, SharedPin } from "@/lib/sharedCategory";
import styles from "./Shared.module.css";

// MapLibre はブラウザでしか動かないので、サーバーでは描画しない（ssr: false はクライアントコンポーネントでのみ使える）
const SharedMap = dynamic(() => import("./SharedMap"), {
  ssr: false,
  loading: () => <div className={styles.message}>地図を読み込んでいます…</div>,
});

export default function SharedMapLoader(props: { category: SharedCategory; pins: SharedPin[] }) {
  return <SharedMap {...props} />;
}
