# pins（ピンメモ地図アプリ）

現在地周辺の地図にピンを立ててメモを書ける Web アプリ。会員登録・ログインなしで、ブラウザからすぐ使えます。

## 主な機能

- 地図上にピンを立ててメモを残す
- ピンをカテゴリ（飲食店・駅など）で色分けし、カテゴリごとに絞り込み表示
- 共有キー：ピンのグループにキーを発行し、相手がキーを入力すると同じピンとメモを表示できる（LINE や X で共有する想定）

## 技術構成

| 領域 | 技術 |
| --- | --- |
| フロントエンド | React + TypeScript、MapLibre GL JS + react-map-gl |
| 地図タイル | [OpenFreeMap](https://openfreemap.org/)（© OpenStreetMap contributors） |
| バックエンド | API Gateway + Lambda + DynamoDB（ap-northeast-1） |
| インフラ | AWS CDK（TypeScript） |
| ホスティング | AWS Amplify Hosting |

## ディレクトリ構成

```
.
├── web/    # フロントエンド
└── infra/  # AWS インフラ（CDK）
```

## ステータス

開発環境の準備中
