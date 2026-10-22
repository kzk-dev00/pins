# api

ピンメモ地図のバックエンド（Go / AWS Lambda / DynamoDB）。仕様はリポジトリ直下の `design.md` を参照。

```
Next.js → API Gateway（HTTP API, ANY /{proxy+}） → Lambda（このコード） → DynamoDB
```

## ファイル構成

| ファイル | 内容 |
|---|---|
| `main.go` | 起動処理。環境変数を読んで DynamoDB に接続する |
| `app.go` | URL の振り分けと各 API の処理（入力チェック、上限、ゲストの期限） |
| `owner.go` | 持ち主の識別（ゲストトークンのハッシュ、会員の JWT の `sub`） |
| `model.go` | DynamoDB に保存する形と、API で返す JSON の形 |
| `store.go` | 保存先のインターフェース |
| `store_dynamo.go` | DynamoDB への読み書き |
| `store_memory.go` | テスト用のメモリ上の保存先 |
| `response.go` | レスポンスとエラー（画面にそのまま出せる日本語のメッセージ） |
| `config.go` | 上限の環境変数の読み込み |
| `util.go` | ランダム ID、入力チェックの補助 |
| `*_test.go` | テスト（AWS にはつながない） |
| `iam-policy.json` | Lambda の実行ロールに付ける DynamoDB の権限（dev 用） |

## コマンド

| コマンド | 内容 |
|---|---|
| `make tidy` | `go.mod` と `go.sum` をコードに合わせて整える |
| `make test` | テストを実行する |
| `make vet` | よくある間違いを静的にチェックする |
| `make build` | Lambda（arm64）用の実行ファイル `bootstrap` を作る |
| `make zip` | アップロード用の `function.zip` を作る |
| `make clean` | `bootstrap` と `function.zip` を消す |

## 環境変数

| 名前 | 必須 | 内容 | 初期値 |
|---|---|---|---|
| `CATEGORIES_TABLE` | ✓ | カテゴリのテーブル名（例：`pin-memo-dev-categories`） | |
| `PINS_TABLE` | ✓ | ピンのテーブル名（例：`pin-memo-dev-pins`） | |
| `GUEST_MAX_CATEGORIES` | | ゲストが作れるカテゴリの数 | 3 |
| `GUEST_MAX_PINS` | | ゲストが1カテゴリに立てられるピンの数 | 10 |
| `MEMBER_MAX_CATEGORIES` | | 会員が作れるカテゴリの数 | 50 |
| `MEMBER_MAX_PINS` | | 会員が1カテゴリに立てられるピンの数 | 1000 |

リージョンは Lambda が自動で設定する `AWS_REGION` を使う。

## Lambda の設定

- ランタイム：`provided.al2023`、アーキテクチャ：`arm64`、ハンドラ：`bootstrap`
- 実行ロール：`iam-policy.json` の権限と、ログ出力用の `AWSLambdaBasicExecutionRole`
- API Gateway（HTTP API）のルートは `ANY /{proxy+}` の1本。ペイロードはバージョン 2.0
