# ピンメモ地図

地図をタップしてピンとメモを残し、カテゴリ単位で共有できる Web アプリ。

**作業の前に必ず `design.md` と `docs/private/ops.md` を読むこと。** `design.md` には技術構成・DynamoDB のテーブル設計・API 仕様・FE の仕様、`docs/private/ops.md` には AWS アカウントの注意点・ゲストの期限（TTL）のルール・上限の値・デプロイしたリソース・進み具合がまとまっている。設計は claude.ai での相談で決めたもので、実装はこれに従う。設計を変えるときは、先にユーザーに確認してから `design.md` か `docs/private/ops.md` も更新する。

`docs/private/ops.md` は GitHub にない非公開のメモで、このPCにだけある（`.gitignore` で除外している）。絶対にコミットしない。

## 構成

- `api/`：BE（Go / AWS Lambda / DynamoDB）。使い方は `api/README.md`
- `web/`：FE（Next.js）。使い方は `web/README.md`

## 作業のルール

- やりとりは日本語
- コミットメッセージは日本語で短く書く（例：「README更新」）
- AWS のリージョンはシドニー（`ap-southeast-2`）。無料プランの制限でシドニーになっている
- AWS リソース名は `pin-memo-<環境>-<用途>`（例：`pin-memo-dev-api`）
- AWS アカウントは新しいシンプル版。「高度な機能を有効にする」に関わる操作はしない
- お金がかかる AWS の操作、リソースの削除、本番（`-prod`）への変更は、実行前にユーザーに確認する
- BE を変えたら `make test` を通す
- 無駄な半角スペースや「、」をいれない
- design.mdには余計な記述を入れない