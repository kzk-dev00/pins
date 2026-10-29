import Link from "next/link";
import AboutContact from "./AboutContact";
import AboutContent from "./AboutContent";
import styles from "../PinMemo/PinMemoMenu.module.css";

// メニューの中で開くページ。null はどれも開いていないとき
export type AboutPage = "about" | "contact" | null;

// 「pinsとは」「お問い合わせ」「プライバシーポリシー」のリンク（自分の地図と共有ページのメニューで共通）
export function AboutMenuLinks({ onOpen, disabled }: { onOpen: (page: AboutPage) => void; disabled?: boolean }) {
  return (
    <section className={styles.section}>
      <button type="button" className={styles.menuLink} onClick={() => onOpen("about")} disabled={disabled}>
        pinsとは
      </button>
      <button type="button" className={styles.menuLink} onClick={() => onOpen("contact")} disabled={disabled}>
        お問い合わせ
      </button>
      <Link className={styles.menuLink} href="/privacy">
        プライバシーポリシー
      </Link>
    </section>
  );
}

// 「pinsとは」「お問い合わせ」のページ。shared は共有ページから開いたとき
export function AboutMenuPage({
  page,
  shared,
  onBack,
}: {
  page: "about" | "contact";
  shared?: boolean;
  onBack: () => void;
}) {
  return (
    <section className={styles.section}>
      <button type="button" className={styles.back} onClick={onBack}>
        ← 戻る
      </button>
      <h3 className={styles.shareTitle}>{page === "about" ? "pinsとは" : "お問い合わせ"}</h3>
      {page === "about" ? <AboutContent shared={shared} /> : <AboutContact />}
    </section>
  );
}
