# web

ピンメモ地図のフロントエンド（Next.js / React / TypeScript）。仕様はリポジトリ直下の `design.md` を参照。

地図は MapLibre GL JS ＋ react-map-gl、地図タイルは OpenFreeMap。API は `api/`（API Gateway → Lambda → DynamoDB）。

## 動かし方

```sh
cp .env.example .env.local   # 初回だけ
npm install
npm run dev
```

ブラウザで http://localhost:3000 を開く。API の CORS は `http://localhost:3000` だけを許可しているので、ポートは変えない。

## コマンド

| コマンド | 内容 |
|---|---|
| `npm run dev` | 開発サーバーを起動する |
| `npm run build` | 本番用にビルドする（型チェックも行う） |
| `npm run lint` | ESLint でチェックする |

## 環境変数

ローカルは `.env.local`、本番は Amplify Hosting の環境変数に設定する。`NEXT_PUBLIC_` の値はビルド時に埋め込まれるので、変えたら再ビルド（再デプロイ）が必要。ブラウザに送られて誰でも見られるので、秘密の値は入れない。

| 名前 | 内容 | 未設定のとき |
|---|---|---|
| `NEXT_PUBLIC_API_BASE_URL` | API Gateway の URL（例は `.env.example`） | エラーを表示する |
| `NEXT_PUBLIC_DEFAULT_LAT` | 地図の初期位置の緯度（現在地が取れないとき） | 東京駅 `35.6812` |
| `NEXT_PUBLIC_DEFAULT_LNG` | 地図の初期位置の経度 | 東京駅 `139.7671` |
| `NEXT_PUBLIC_DEFAULT_ZOOM` | 地図の初期の拡大率（0〜22） | `14` |

## ファイル構成

| ファイル | 内容 |
|---|---|
| `src/app/page.tsx` | トップページ。地図を表示するだけ |
| `src/components/PinMemo/MapLoader.tsx` | 地図をブラウザでだけ読み込む（`ssr: false`） |
| `src/components/PinMemo/Map.tsx` | 地図、ピン、メモ入力シート |
| `src/components/PinMemo/Map.module.css` | 地図まわりのスタイル |
| `src/components/PinMemo/Menu.tsx` | 右上の ☰ から開くメニュー（カテゴリ一覧・追加） |
| `src/components/PinMemo/Menu.module.css` | メニューのスタイル |
| `src/components/PinMemo/CategoryPicker.tsx` | メモのシートでカテゴリを選ぶ欄（複数可） |
| `src/components/PinMemo/CategoryPicker.module.css` | カテゴリを選ぶ欄のスタイル |
| `src/components/PinMemo/SharePanel.tsx` | メニューの中の共有画面（URL の作成・コピー・作り直し・共有をやめる） |
| `src/components/PinMemo/useSheetHeight.ts` | 下から出るシートの高さを測る（帰属表示を持ち上げるため） |
| `src/app/share/[viewKey]/page.tsx` | 共有ページ。サーバー側でデータを取り、OGP のタグを出す |
| `src/components/Share/SharedMapLoader.tsx` | 共有ページの地図をブラウザでだけ読み込む |
| `src/components/Share/SharedMap.tsx` | 共有ページの地図（見るだけ。ピンが収まる初期位置） |
| `src/components/Share/Shared.module.css` | 共有ページのスタイル |
| `src/lib/sharedCategory.ts` | 共有ページのデータの取得（GSI の反映遅れに備えて1回だけ再試行） |
| `src/lib/api.ts` | API の呼び出し |
| `src/lib/guestToken.ts` | ゲストトークン（localStorage に保存） |
| `src/lib/config.ts` | 環境変数から読む設定（地図の初期位置） |
| `src/lib/lastView.ts` | 最後に見ていた地図の位置（localStorage に保存） |
| `src/lib/hiddenCategories.ts` | 表示を OFF にしたカテゴリの ID（localStorage に保存） |
| `src/lib/categoryColors.ts` | カテゴリの8色と、次に使う色 |
| `scripts/copy-maplibre-worker.mjs` | MapLibre の Web Worker を `public/` にコピーする。`npm run dev` / `build` の前に自動で実行される |

MapLibre は Worker を本体のファイルの隣から読もうとするが、Next.js（Turbopack）でまとめるとそこに Worker がなくなり、地図が描画されない。そのため `public/` から配信し、`setWorkerUrl` で場所を指定している。

## 今の制限

- 「pins とは」（アプリの説明）はまだない
- ゲストは1カテゴリにピン10件まで（API 側の上限）
