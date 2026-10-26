"use client";

import { useEffect, useState } from "react";
import AboutContent from "../About/AboutContent";
import styles from "../PinMemo/PinMemoMenu.module.css";

// 共有ページの ☰ メニュー。見るだけのページなので、中身は「pins とは」だけ
export default function SharedMenu({ onClose }: { onClose: () => void }) {
  const [aboutOpen, setAboutOpen] = useState(false);

  // Esc で閉じる（「pins とは」を開いているときはメニューに戻る）
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      if (aboutOpen) setAboutOpen(false);
      else onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [aboutOpen, onClose]);

  return (
    <div className={styles.backdrop} onClick={onClose}>
      <aside
        className={styles.panel}
        role="dialog"
        aria-modal="true"
        aria-labelledby="shared-menu-title"
        onClick={(e) => e.stopPropagation()}
      >
        <header className={styles.header}>
          <h2 id="shared-menu-title" className={styles.title}>
            メニュー
          </h2>
          <button type="button" className={styles.close} onClick={onClose} aria-label="メニューを閉じる">
            ×
          </button>
        </header>

        {aboutOpen ? (
          <section className={styles.section}>
            <button type="button" className={styles.back} onClick={() => setAboutOpen(false)}>
              ← メニューに戻る
            </button>
            <h3 className={styles.shareTitle}>pins とは</h3>
            <AboutContent shared />
          </section>
        ) : (
          <section className={styles.section}>
            <button type="button" className={styles.menuLink} onClick={() => setAboutOpen(true)}>
              pins とは
            </button>
          </section>
        )}
      </aside>
    </div>
  );
}
