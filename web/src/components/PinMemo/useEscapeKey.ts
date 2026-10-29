"use client";

import { useEffect, useRef } from "react";

// enabled のあいだ、Esc を押したら onEscape を呼ぶ（自分の地図・共有ページ・メニューで共通）
export function useEscapeKey(onEscape: () => void, enabled = true) {
  // 毎回の描画で変わる onEscape を、登録し直さずに最新のものを呼べるようにする
  const handlerRef = useRef(onEscape);
  useEffect(() => {
    handlerRef.current = onEscape;
  });

  useEffect(() => {
    if (!enabled) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") handlerRef.current();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [enabled]);
}
