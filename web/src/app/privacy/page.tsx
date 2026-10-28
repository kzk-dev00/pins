import type { Metadata } from "next";
import Link from "next/link";
import styles from "./privacy.module.css";

export const metadata: Metadata = {
  title: "プライバシーポリシー | pins",
  description: "pinsのプライバシーポリシー",
};

const CONTACT_EMAIL = "pins.servicejp@gmail.com";

// プライバシーポリシー。内容はdesign.mdの仕様（保存するデータ・期限・共有）に合わせる
export default function PrivacyPage() {
  return (
    <main className={styles.page}>
      <h1 className={styles.title}>プライバシーポリシー</h1>
      <p>
        pins（以下「本アプリ」）は、利用者の情報を以下のとおり取り扱います。
      </p>

      <h2 className={styles.heading}>1.保存する情報</h2>
      <p>本アプリは、利用者が入力した次の情報をサーバーに保存します。</p>
      <ul className={styles.list}>
        <li>ピンの位置（緯度・経度）、タイトル、メモ</li>
        <li>カテゴリの名前・色・並び順、共有の設定</li>
        <li>利用者を見分けるための値（下記「3.利用者の識別」を参照）</li>
      </ul>
      <p>ピンの位置は、利用者が地図上で選んだ場所です。自宅などの場所にピンを立てると、その場所が保存されます。</p>

      <h2 className={styles.heading}>2.現在地（位置情報）</h2>
      <p>
        現在地は地図を現在地へ移動するために取得しお使いの端末の中でのみ使います。現在地をサーバーへ送ったり保存したりすることはありません。
      </p>

      <h2 className={styles.heading}>3.利用者の識別</h2>
      <p>
        本アプリは会員登録なしで使えます。
      </p>
      <p>
        この他、最後に見ていた地図の位置や表示を切り替えたカテゴリなどの設定を、お使いのブラウザに保存しそれらはサーバーへ送りません。
      </p>

      <h2 className={styles.heading}>4.情報を使う目的</h2>
      <ul className={styles.list}>
        <li>ピンやメモを保存し表示するため</li>
        <li>利用者が共有したカテゴリを共有URLを開いた人に表示するため</li>
        <li>不具合の調査やお問い合わせへの対応のため</li>
      </ul>

      <h2 className={styles.heading}>5.共有について</h2>
      <p>
        カテゴリを共有すると共有URLを知っている人は誰でもそのカテゴリのピンの位置・タイトル・メモを見られます。SNSなどにURLを投稿すると、多くの人に見られる可能性があります。共有はいつでもやめられ、やめると共有URLは使えなくなります。
      </p>

      <h2 className={styles.heading}>6.保存期間</h2>
      <p>
        保存した情報は、しばらく（30日ほど）使わないと自動で削除されます。またブラウザのデータを消すとそれまでの情報は削除されます。
      </p>

      <h2 className={styles.heading}>7.第三者への提供</h2>
      <p>
        法令に基づく場合を除き、利用者の情報を第三者に提供することはありません（利用者自身が共有したカテゴリを除きます）。
      </p>

      <h2 className={styles.heading}>8.お問い合わせ</h2>
      <p>
        本ポリシーや、保存された情報についてのお問い合わせは、
        <a className={styles.link} href={`mailto:${CONTACT_EMAIL}`}>
          {CONTACT_EMAIL}
        </a>
        までご連絡ください。
      </p>

      <h2 className={styles.heading}>9.免責事項</h2>
      <p>本アプリは個人開発のものです。利用により生じたいかなる損害についても、作成者は責任を負いかねますので、あらかじめご了承ください。</p>

      <h2 className={styles.heading}>10.改定</h2>
      <p>本ポリシーは、必要に応じて改定することがあります。改定した場合は、このページでお知らせします。</p>

      <p className={styles.date}>2026年10月27日制定</p>

      <Link className={styles.back} href="/">
        ← 戻る
      </Link>
    </main>
  );
}
