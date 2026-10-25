import styles from "./About.module.css";

type Props = {
  // 共有ページで表示するとき true。「見るだけ」の説明を先頭に出す
  shared?: boolean;
};

// 「pins とは」の本文。自分の地図と共有ページのメニューで使う
export default function AboutContent({ shared = false }: Props) {
  return (
    <div className={styles.about}>
      {shared && (
        <p className={styles.lead}>
          このページは他の誰かが共有したカテゴリの地図です。ピンとメモを見ることだけができます。
        </p>
      )}

      <p>pinsは地図をタップしてピンを立てメモを残せるアプリです。気になるお店や行きたい場所をカテゴリにまとめ手軽に共有できます。</p>

      <h4 className={styles.heading}>使い方</h4>
      <ol className={styles.list}>
        <li>地図をタップするとピンが立ちます。メモとカテゴリを入れて保存します。</li>
        <li>ピンをタップするとメモやカテゴリを変えられます。ピンはドラッグで動かせます。</li>
        <li>右上の ☰ から、カテゴリの追加・名前の変更・表示の切り替えができます。</li>
        <li>カテゴリごとに共有URLを作って他の人と共有できます。※共有した側からいつでも無効にすることが可能です</li>
      </ol>

      <h4 className={styles.heading}>会員登録なしで使えます</h4>
      <ul className={styles.list}>
        <li>データは、お使いの端末のブラウザに保存され他の端末やブラウザからは見られません。</li>
        <li>ブラウザのデータを消すと、見られなくなります。</li>
        <li>しばらく（30日ほど）使わないと自動で消えます。</li>
        <li>カテゴリは3つまで、1つのカテゴリにピンは10件まで作れます。</li>
      </ul>

      <p className={styles.credit}>
        地図：© OpenStreetMap contributors / OpenFreeMap / MapLibre
      </p>
    </div>
  );
}
