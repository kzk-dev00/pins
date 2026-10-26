// MapLibre の Web Worker を public/ にコピーする。
// MapLibre は Worker の場所を本体のファイルの隣だと決めつけるが、Next.js（Turbopack）でまとめると
// そこに Worker がなくなるため、public/ から配信して setWorkerUrl で場所を教える。
import { copyFileSync, mkdirSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const dist = join(dirname(require.resolve("maplibre-gl/package.json")), "dist");
const publicDir = join(dirname(fileURLToPath(import.meta.url)), "..", "public");

// public/ に入るのはこのコピーだけで、.gitignore で除外しているため、Git から取ってきた直後（Amplify のビルドなど）は
// public/ 自体がない。そのためmkdirSyncで作成する
mkdirSync(publicDir, { recursive: true });
copyFileSync(join(dist, "maplibre-gl-worker.mjs"), join(publicDir, "maplibre-gl-worker.mjs"));
