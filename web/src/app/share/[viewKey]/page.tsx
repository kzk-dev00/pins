import type { Metadata } from "next";
import { Suspense } from "react";
import SharedMapLoader from "@/components/Share/SharedMapLoader";
import { fetchSharedCategory } from "@/lib/sharedCategory";
import styles from "@/components/Share/Shared.module.css";

const APP_NAME = "pins";

// LINE や X に URL を貼ったときのプレビュー（OGP）に、カテゴリ名を出す。
// 共有ページは URL を知っている人だけに見せるものなので、検索エンジンには載せない
export async function generateMetadata({ params }: PageProps<"/share/[viewKey]">): Promise<Metadata> {
  const { viewKey } = await params;
  const result = await fetchSharedCategory(viewKey);
  const robots = { index: false, follow: false };
  if (result.status !== "ok") {
    return { title: `共有されたカテゴリが見つかりません | ${APP_NAME}`, robots };
  }
  const { category, pins } = result.data;
  const title = `${category.name} | ${APP_NAME}`;
  const description = `「${category.name}」のピン${pins.length}件とメモを地図で見られます`;
  return {
    title,
    description,
    robots,
    openGraph: { title, description, type: "website", siteName: APP_NAME },
    twitter: { card: "summary", title, description },
  };
}

// URL の viewKey はリクエストのたびに決まる値なので、Suspense で囲んで読む（cacheComponents の決まり）
export default function SharePage({ params }: PageProps<"/share/[viewKey]">) {
  return (
    <Suspense fallback={<div className={styles.message}>読み込んでいます…</div>}>
      {params.then(({ viewKey }) => (
        <SharedContent viewKey={viewKey} />
      ))}
    </Suspense>
  );
}

async function SharedContent({ viewKey }: { viewKey: string }) {
  const result = await fetchSharedCategory(viewKey);
  if (result.status === "not_found") {
    return (
      <div className={styles.message} role="alert">
        共有されたカテゴリが見つかりません。共有が停止されたか、期限が切れた可能性があります。
      </div>
    );
  }
  if (result.status === "error") {
    return (
      <div className={styles.message} role="alert">
        読み込めませんでした。時間をおいて、もう一度開いてください。
      </div>
    );
  }
  return <SharedMapLoader category={result.data.category} pins={result.data.pins} />;
}
