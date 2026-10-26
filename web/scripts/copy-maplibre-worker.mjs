// MapLibre の Web Worker を public/ にコピーする。
// MapLibre は Worker の場所を本体のファイルの隣だと決めつけるが、Next.js（Turbopack）でまとめると
// そこに Worker がなくなるため、public/ から配信して setWorkerUrl で場所を教える。
import { copyFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const dist = join(dirname(require.resolve("maplibre-gl/package.json")), "dist");
const publicDir = join(dirname(fileURLToPath(import.meta.url)), "..", "public");

// コピーしたファイルは .gitignore で除外している。public/ がほかに空でも Git に残るよう、public/.gitkeep を置いている
// （Git は空のフォルダを記録しないため。public/ にほかのファイルを置いたら .gitkeep は消してよい）
copyFileSync(join(dist, "maplibre-gl-worker.mjs"), join(publicDir, "maplibre-gl-worker.mjs"));
