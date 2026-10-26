"use client";

import { useCallback, useState } from "react";

// ピンが重なったとき、最後に触ったピンを一番上に出すための順番を覚える。
// 新しく触ったものほど上。再読み込みすると元の順番に戻る（保存はしない）
export function usePinFront() {
  // 触った順のピンID（最後が一番上）
  const [order, setOrder] = useState<string[]>([]);

  const bringToFront = useCallback((pinId: string) => {
    setOrder((prev) => (prev.at(-1) === pinId ? prev : [...prev.filter((id) => id !== pinId), pinId]));
  }, []);

  // Marker の重なり順（z-index）。触ったことのないピンは 0 で、地図に追加された順のまま
  const zIndexOf = (pinId: string) => order.indexOf(pinId) + 1;

  return { bringToFront, zIndexOf, frontZIndex: order.length + 1 };
}
