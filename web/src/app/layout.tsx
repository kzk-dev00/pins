import type { Metadata } from "next";
import { Zen_Kaku_Gothic_New } from "next/font/google";
import "./globals.css";

const zenKakuGothicNew = Zen_Kaku_Gothic_New({
  variable: "--font-zen-kaku",
  weight: ["400", "700"],
  subsets: ["latin"],
  preload: false, // 日本語フォントはファイルが大きいので先読みしない
});

export const metadata: Metadata = {
  title: "pins",
  description: "地図をタップしてピンとメモを残せるアプリ",
};

// viewport-fit=cover は使わない。iOS 26 の Safari ではページがアドレスバーの裏まで広がり、
// 右上の ☰ ボタンなどがバーに隠れてしまうため（env(safe-area-inset-*) にはバーの高さが入らない）

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="ja" className={zenKakuGothicNew.variable}>
      <body>{children}</body>
    </html>
  );
}
