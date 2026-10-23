# AI エージェント向けの注意（web/）

## 使用されている Next.js のバージョンについて

AI の学習データより新しく、API や書き方が変わっている可能性がある。コードを書く前に `web/node_modules/next/dist/docs/` の該当ガイドを読み、非推奨の注意に従うこと。

## このファイルについて

もとは Next.js が自動で作る英語の注意書きだった。日本語で管理するため、`next.config.ts` の `agentRules: false` で自動の追加・書き戻しを止めている。Next.js を大きく更新したときは、`node_modules/next/dist/docs/01-app/02-guides/ai-agents.md` を見て、注意書きの内容が変わっていないか確認する。
