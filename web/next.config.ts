import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  // agentRules: falseについて
  // AGENTS.md は日本語で自分たちで管理するため、next dev が英語の注意書き
  // （nextjs-agent-rules の部分）を自動で追加・書き戻さないようにする。
  // 内容は「同梱のドキュメント node_modules/next/dist/docs/ を読む」で、日本語の AGENTS.md に同じことを書いている
  agentRules: false,
  cacheComponents: true,
  partialPrefetching: true,
};

export default nextConfig;
