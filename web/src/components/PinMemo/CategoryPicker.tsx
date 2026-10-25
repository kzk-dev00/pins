"use client";

import { useState } from "react";
import type { Category } from "@/lib/api";
import styles from "./CategoryPicker.module.css";

type Props = {
  categories: Category[];
  selectedIds: string[];
  onChange: (categoryIds: string[]) => void;
  disabled?: boolean;
};

// メモのシートの「カテゴリ」欄。タップで一覧を開き、チェックボックスで選ぶ（複数可）
export default function CategoryPicker({ categories, selectedIds, onChange, disabled }: Props) {
  const [open, setOpen] = useState(false);
  const selected = categories.filter((c) => selectedIds.includes(c.categoryId));

  const toggle = (categoryId: string) => {
    const next = selectedIds.includes(categoryId)
      ? selectedIds.filter((id) => id !== categoryId)
      : [...selectedIds, categoryId];
    // 一覧の並び順（sortOrder 順）にそろえて渡す
    onChange(categories.filter((c) => next.includes(c.categoryId)).map((c) => c.categoryId));
  };

  return (
    <div className={styles.root}>
      <button
        type="button"
        className={styles.field}
        onClick={() => setOpen((v) => !v)}
        disabled={disabled}
        aria-expanded={open}
        aria-controls="category-picker-list"
      >
        <span className={styles.value}>
          {selected.length === 0 ? (
            <span className={styles.placeholder}>カテゴリを選んでください</span>
          ) : (
            selected.map((c) => (
              <span key={c.categoryId} className={styles.chip}>
                <span className={styles.dot} style={{ background: c.color }} aria-hidden />
                {c.name}
              </span>
            ))
          )}
        </span>
        <span className={styles.chevron} aria-hidden>
          {open ? "▲" : "▼"}
        </span>
      </button>

      {open && (
        <ul id="category-picker-list" className={styles.list}>
          {categories.length === 0 && (
            <li className={styles.empty}>カテゴリがありません。右上のメニューから追加してください。</li>
          )}
          {categories.map((c) => (
            <li key={c.categoryId}>
              <label className={styles.option}>
                <input
                  type="checkbox"
                  checked={selectedIds.includes(c.categoryId)}
                  onChange={() => toggle(c.categoryId)}
                  disabled={disabled}
                />
                <span className={styles.dot} style={{ background: c.color }} aria-hidden />
                <span className={styles.name}>{c.name}</span>
              </label>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
