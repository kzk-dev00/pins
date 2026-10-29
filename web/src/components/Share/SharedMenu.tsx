"use client";

import { useState } from "react";
import { AboutMenuLinks, AboutMenuPage, type AboutPage } from "../About/AboutMenu";
import { useEscapeKey } from "../PinMemo/useEscapeKey";
import styles from "../PinMemo/PinMemoMenu.module.css";

// 共有ページの ☰ メニュー。見るだけのページなので、中身は「pinsとは」「お問い合わせ」「プライバシーポリシー」だけ
export default function SharedMenu({ onClose }: { onClose: () => void }) {
  const [page, setPage] = useState<AboutPage>(null);

  // Esc で閉じる（「pinsとは」「お問い合わせ」を開いているときはメニューに戻る）
  useEscapeKey(() => {
    if (page) setPage(null);
    else onClose();
  });

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

        {page ? <AboutMenuPage page={page} shared onBack={() => setPage(null)} /> : <AboutMenuLinks onOpen={setPage} />}
      </aside>
    </div>
  );
}
