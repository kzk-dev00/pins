"use client";

import { useEffect, useRef, useState } from "react";

// 下から出るシートの高さを測る。開いているあいだ、右下の帰属表示（© OpenStreetMap など）を
// この分だけ持ち上げて、シートのボタンと重ならないようにするために使う
export function useSheetHeight<T extends HTMLElement>(open: boolean) {
  const ref = useRef<T>(null);
  const [height, setHeight] = useState(0);

  useEffect(() => {
    const el = ref.current;
    if (!open || !el) return;
    const observer = new ResizeObserver(() => setHeight(el.offsetHeight));
    observer.observe(el);
    return () => observer.disconnect();
  }, [open]);

  return [ref, height] as const;
}
