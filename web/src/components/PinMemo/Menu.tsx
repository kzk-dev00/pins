"use client";

import { useEffect, useState, type FormEvent } from "react";
import type { Category } from "@/lib/api";
import SharePanel from "./SharePanel";
import styles from "./Menu.module.css";

const CATEGORY_NAME_MAX_LENGTH = 30;

// API と同じく、前後の空白を除いて1〜30文字（文字数はコードポイントで数える）
function isValidCategoryName(name: string): boolean {
  const length = [...name.trim()].length;
  return length >= 1 && length <= CATEGORY_NAME_MAX_LENGTH;
}

type Props = {
  categories: Category[];
  hiddenIds: ReadonlySet<string>;
  busy: boolean;
  error: string | null;
  onToggle: (categoryId: string) => void;
  // 成功したら true（入力欄を空に戻す・編集を閉じるため）
  onCreate: (name: string) => Promise<boolean>;
  onRename: (categoryId: string, name: string) => Promise<boolean>;
  onDelete: (category: Category) => void;
  onShare: (categoryId: string) => Promise<boolean>;
  onUnshare: (categoryId: string) => Promise<boolean>;
  onClose: () => void;
};

// 右上の ☰ から開くメニュー。カテゴリ一覧（ON/OFF・名前の変更・削除・共有）と、新規カテゴリの追加
export default function Menu({
  categories,
  hiddenIds,
  busy,
  error,
  onToggle,
  onCreate,
  onRename,
  onDelete,
  onShare,
  onUnshare,
  onClose,
}: Props) {
  const [newName, setNewName] = useState("");
  const [renaming, setRenaming] = useState<{ categoryId: string; name: string } | null>(null);
  // 共有画面を開いているカテゴリ
  const [sharingId, setSharingId] = useState<string | null>(null);
  const sharing = categories.find((c) => c.categoryId === sharingId) ?? null;

  // Esc で閉じる（名前の変更中は変更だけを、共有画面ではカテゴリ一覧に戻る）
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== "Escape" || busy) return;
      if (renaming) setRenaming(null);
      else if (sharingId) setSharingId(null);
      else onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [busy, renaming, sharingId, onClose]);

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (busy || !isValidCategoryName(newName)) return;
    if (await onCreate(newName.trim())) setNewName("");
  };

  const handleRename = async (e: FormEvent) => {
    e.preventDefault();
    if (busy || !renaming || !isValidCategoryName(renaming.name)) return;
    if (await onRename(renaming.categoryId, renaming.name.trim())) setRenaming(null);
  };

  return (
    <div className={styles.backdrop} onClick={busy ? undefined : onClose}>
      <aside
        className={styles.panel}
        role="dialog"
        aria-modal="true"
        aria-labelledby="menu-title"
        onClick={(e) => e.stopPropagation()}
      >
        <header className={styles.header}>
          <h2 id="menu-title" className={styles.title}>
            メニュー
          </h2>
          <button type="button" className={styles.close} onClick={onClose} disabled={busy} aria-label="メニューを閉じる">
            ×
          </button>
        </header>

        {error && (
          <p className={styles.error} role="alert">
            {error}
          </p>
        )}

        {sharing ? (
          <SharePanel
            category={sharing}
            busy={busy}
            onShare={onShare}
            onUnshare={onUnshare}
            onBack={() => setSharingId(null)}
          />
        ) : (
          <>
            <section className={styles.section}>
              <h3 className={styles.sectionTitle}>カテゴリ</h3>
              {categories.length === 0 ? (
                <p className={styles.empty}>カテゴリがありません。下から追加してください。</p>
              ) : (
                <ul className={styles.list}>
                  {categories.map((category) => (
                    <li key={category.categoryId} className={styles.item}>
                      {renaming?.categoryId === category.categoryId ? (
                        <form className={styles.renameForm} onSubmit={handleRename}>
                          <input
                            className={styles.input}
                            value={renaming.name}
                            onChange={(e) => setRenaming({ ...renaming, name: e.target.value })}
                            maxLength={CATEGORY_NAME_MAX_LENGTH}
                            aria-label="新しいカテゴリ名"
                            autoFocus
                          />
                          <button
                            type="submit"
                            className={`${styles.button} ${styles.primary}`}
                            disabled={busy || !isValidCategoryName(renaming.name)}
                          >
                            保存
                          </button>
                          <button type="button" className={styles.button} onClick={() => setRenaming(null)} disabled={busy}>
                            やめる
                          </button>
                        </form>
                      ) : (
                        <>
                          <label className={styles.toggle}>
                            <input
                              type="checkbox"
                              checked={!hiddenIds.has(category.categoryId)}
                              onChange={() => onToggle(category.categoryId)}
                            />
                            <span className={styles.dot} style={{ background: category.color }} aria-hidden />
                            <span className={styles.name}>{category.name}</span>
                          </label>
                          <button
                            type="button"
                            className={styles.button}
                            onClick={() => setRenaming({ categoryId: category.categoryId, name: category.name })}
                            disabled={busy}
                            aria-label={`「${category.name}」の名前を変更`}
                          >
                            名前
                          </button>
                          <button
                            type="button"
                            className={`${styles.button} ${category.viewKey ? styles.sharing : ""}`}
                            onClick={() => setSharingId(category.categoryId)}
                            disabled={busy}
                            aria-label={`「${category.name}」を共有${category.viewKey ? "（共有中）" : ""}`}
                          >
                            {category.viewKey ? "共有中" : "共有"}
                          </button>
                          <button
                            type="button"
                            className={`${styles.button} ${styles.danger}`}
                            onClick={() => onDelete(category)}
                            disabled={busy}
                            aria-label={`「${category.name}」を削除`}
                          >
                            削除
                          </button>
                        </>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section className={styles.section}>
              <h3 className={styles.sectionTitle}>新規カテゴリの追加</h3>
              <form className={styles.createForm} onSubmit={handleCreate}>
                <input
                  className={styles.input}
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="例：厚木ランチ"
                  maxLength={CATEGORY_NAME_MAX_LENGTH}
                  aria-label="新しいカテゴリ名"
                />
                <button
                  type="submit"
                  className={`${styles.button} ${styles.primary}`}
                  disabled={busy || !isValidCategoryName(newName)}
                >
                  追加
                </button>
              </form>
              <p className={styles.counter}>
                {[...newName.trim()].length}/{CATEGORY_NAME_MAX_LENGTH}
              </p>
            </section>
          </>
        )}
      </aside>
    </div>
  );
}
