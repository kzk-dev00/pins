import type { CSSProperties } from "react";
import styles from "./PinMemoMap.module.css";

type Props = {
  color?: string;
  // ピンの真上に出す見出し。空ならラベルを出さない
  title?: string;
  // 保存前の下書き（半透明）
  draft?: boolean;
  // マウスを乗せたときに出す説明
  tooltip?: string;
};

// 地図に立てるピンの見た目。自分の地図と共有ページで使う。
// ラベルはピンの上に重ねて置く（ピンの大きさを変えないので、Marker の位置がずれない）
export default function PinMemoPinIcon({ color, title, draft = false, tooltip }: Props) {
  return (
    <div className={styles.marker} title={tooltip}>
      {title && <span className={styles.label}>{title}</span>}
      <div
        className={`${styles.pin} ${draft ? styles.draft : ""}`}
        style={{ "--pin-color": color } as CSSProperties}
      />
    </div>
  );
}
