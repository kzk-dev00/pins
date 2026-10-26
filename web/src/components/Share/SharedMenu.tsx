"use client";

import { useEffect, useState } from "react";
import AboutContact from "../About/AboutContact";
import AboutContent from "../About/AboutContent";
import styles from "../PinMemo/PinMemoMenu.module.css";

// 共有ページの ☰ メニュー。見るだけのページなので、中身は「pins とは」と「お問い合わせ」だけ
export default function SharedMenu({ onClose }: { onClose: () => void }) {
  const [page, setPage] = useState<"about" | "contact" | null>(null);

  // Esc で閉じる（「pins とは」「お問い合わせ」を開いているときはメニューに戻る）
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      if (page) setPage(null);
      else onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [page, onClose]);

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

        {page ? (
          <section className={styles.section}>
            <button type="button" className={styles.back} onClick={() => setPage(null)}>
              ← メニューに戻る
            </button>
            <h3 className={styles.shareTitle}>{page === "about" ? "pins とは" : "お問い合わせ"}</h3>
            {page === "about" ? <AboutContent shared /> : <AboutContact />}
          </section>
        ) : (
          <section className={styles.section}>
            <button type="button" className={styles.menuLink} onClick={() => setPage("about")}>
              pins とは
            </button>
            <button type="button" className={styles.menuLink} onClick={() => setPage("contact")}>
              お問い合わせ
            </button>
          </section>
        )}
      </aside>
    </div>
  );
}
