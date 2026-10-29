import { CONTACT_EMAIL } from "@/lib/contact";
import styles from "./About.module.css";

// 「お問い合わせ」の本文。アドレスを押すと、端末のメールアプリが開く（mailto: のリンク）
export default function AboutContact() {
  return (
    <div className={styles.about}>
      <p>ご意見・不具合のご報告などは下記のアドレスまでメールでお送りください。</p>
      <a className={styles.mail} href={`mailto:${CONTACT_EMAIL}`}>
        {CONTACT_EMAIL}
      </a>
    </div>
  );
}
