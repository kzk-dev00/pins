"use client";

import { useState } from "react";
import type { Category } from "@/lib/api";
import styles from "./PinMemoMenu.module.css";

type Props = {
  category: Category;
  busy: boolean;
  // 成功したら true
  onShare: (categoryId: string) => Promise<boolean>;
  onUnshare: (categoryId: string) => Promise<boolean>;
  onBack: () => void;
};

export function shareUrl(viewKey: string): string {
  return `${window.location.origin}/share/${viewKey}`;
}

// メニューの中で開く、カテゴリの共有画面
export default function PinMemoSharePanel({ category, busy, onShare, onUnshare, onBack }: Props) {
  const [copied, setCopied] = useState(false);
  const url = category.viewKey ? shareUrl(category.viewKey) : null;

  const handleCreate = async () => {
    setCopied(false);
    await onShare(category.categoryId);
  };

  const handleCopy = async () => {
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
    } catch {
      setCopied(false);
      window.prompt("コピーできませんでした。この URL を選んでコピーしてください", url);
    }
  };

  return (
    <section className={styles.section}>
      <button type="button" className={styles.back} onClick={onBack} disabled={busy}>
        ← 戻る
      </button>
      <h3 className={styles.shareTitle}>
        <span className={styles.dot} style={{ background: category.color }} aria-hidden />
        「{category.name}」を共有
      </h3>
      <p className={styles.note}>
        URLから該当カテゴリのピンとメモを共有できます（共有された相手は「メモ、ピンの編集はできません」）<br />
        ※共有されたURLはいつでも無効にすることが可能です
      </p>
      <p className={styles.warning}>SNSなどに投稿すると誰でも見られるようになりますのでご注意ください</p>

      {url ? (
        <>
          <input className={styles.input} value={url} readOnly aria-label="共有 URL" onFocus={(e) => e.target.select()} />
          <div className={styles.actions}>
            <button type="button" className={`${styles.button} ${styles.primary}`} onClick={handleCopy} disabled={busy}>
              {copied ? "コピーしました" : "URL をコピー"}
            </button>
          </div>
          <div className={styles.actions}>
            <button
              type="button"
              className={`${styles.button} ${styles.danger}`}
              onClick={() => onUnshare(category.categoryId)}
              disabled={busy}
            >
              共有をやめる
            </button>
          </div>
        </>
      ) : (
        <div className={styles.actions}>
          <button type="button" className={`${styles.button} ${styles.primary}`} onClick={handleCreate} disabled={busy}>
            共有 URL を作る
          </button>
        </div>
      )}
    </section>
  );
}
