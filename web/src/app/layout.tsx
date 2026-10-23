import type { Metadata, Viewport } from "next";
import { Zen_Kaku_Gothic_New } from "next/font/google";
import "./globals.css";

const zenKakuGothicNew = Zen_Kaku_Gothic_New({
  variable: "--font-zen-kaku",
  weight: ["400", "700"],
  subsets: ["latin"],
  preload: false, // 日本語フォントはファイルが大きいので先読みしない
});

export const metadata: Metadata = {
  title: "ピンメモ地図",
  description: "地図をタップしてピンとメモを残せるアプリ",
};

export const viewport: Viewport = {
  viewportFit: "cover", // iPhone の画面下の余白（safe-area）まで使う
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="ja" className={zenKakuGothicNew.variable}>
      <body>{children}</body>
    </html>
  );
}
