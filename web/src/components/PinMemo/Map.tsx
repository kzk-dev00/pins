"use client";

import { useEffect, useMemo, useRef, useState, type CSSProperties, type FormEvent } from "react";
import Map, {
  GeolocateControl,
  Marker,
  NavigationControl,
  type GeolocateControlInstance,
  type MapLayerMouseEvent,
} from "react-map-gl/maplibre";
import { setWorkerUrl } from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import * as api from "@/lib/api";
import type { Category, Pin } from "@/lib/api";
import { nextCategoryColor } from "@/lib/categoryColors";
import { DEFAULT_VIEW } from "@/lib/config";
import { loadHiddenCategoryIds, saveHiddenCategoryIds } from "@/lib/hiddenCategories";
import { loadLastView, saveLastView } from "@/lib/lastView";
import Menu from "./Menu";
import styles from "./Map.module.css";

// Worker は public/ から配信する（scripts/copy-maplibre-worker.mjs がコピーしている）
setWorkerUrl("/maplibre-gl-worker.mjs");

const MAP_STYLE = "https://tiles.openfreemap.org/styles/liberty";
// 初めて開いたとき、カテゴリがなければ自動で作る
const DEFAULT_CATEGORY_NAME = "マイピン";
const MEMO_MAX_LENGTH = 200;

// 編集中のピン。pin が null なら新規（まだ保存していない下書き）
type Editing = { pin: Pin | null; lat: number; lng: number; memo: string };

type Status = "loading" | "ready" | "error";

// StrictMode で effect が2回走っても、カテゴリが2つ作られないように1回にまとめる
let initialLoad: Promise<{ categories: Category[]; pins: Pin[] }> | null = null;

function loadInitialData() {
  initialLoad ??= (async () => {
    let categories = await api.listCategories();
    if (categories.length === 0) {
      categories = [await api.createCategory({ name: DEFAULT_CATEGORY_NAME, color: nextCategoryColor(0) })];
    }
    // 全部のピンを1回で読み込み、カテゴリの ON/OFF は画面側で絞り込む
    const pins = await api.listPins();
    return { categories, pins };
  })();
  initialLoad.catch(() => {
    initialLoad = null; // 失敗したら、次に開いたときにやり直せるようにする
  });
  return initialLoad;
}

export default function PinMemoMap() {
  const [status, setStatus] = useState<Status>("loading");
  const [categories, setCategories] = useState<Category[]>([]);
  const [pins, setPins] = useState<Pin[]>([]);
  // 表示を OFF にしたカテゴリの ID（localStorage に保存）
  const [hiddenIds, setHiddenIds] = useState(loadHiddenCategoryIds);
  const [editing, setEditing] = useState<Editing | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const geolocateRef = useRef<GeolocateControlInstance>(null);
  // 前回見ていた位置。初回（保存なし）は null
  const [lastView] = useState(loadLastView);
  // シートの高さ。開いているあいだ、右下の帰属表示をこの分だけ持ち上げてボタンと重ならないようにする
  const sheetRef = useRef<HTMLFormElement>(null);
  const [sheetHeight, setSheetHeight] = useState(0);
  const sheetOpen = editing !== null;

  const hiddenSet = useMemo(() => new Set(hiddenIds), [hiddenIds]);
  const categoryById = useMemo(() => new globalThis.Map(categories.map((c) => [c.categoryId, c])), [categories]);
  const visiblePins = pins.filter((p) => !hiddenSet.has(p.categoryId));
  // 新しいピンを入れるカテゴリ。メモのシートでカテゴリを選べるようになるまでは、ON のうち一番上のカテゴリ
  const targetCategory = categories.find((c) => !hiddenSet.has(c.categoryId)) ?? categories[0];

  useEffect(() => {
    let cancelled = false;
    loadInitialData()
      .then((data) => {
        if (cancelled) return;
        setCategories(data.categories);
        setPins(data.pins);
        setStatus("ready");
      })
      .catch((err) => {
        if (cancelled) return;
        setError(api.errorMessage(err));
        setStatus("error");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const sheet = sheetRef.current;
    if (!sheetOpen || !sheet) return;
    const observer = new ResizeObserver(() => setSheetHeight(sheet.offsetHeight));
    observer.observe(sheet);
    return () => observer.disconnect();
  }, [sheetOpen]);

  // Esc でシートを閉じる（メニューを開いているときはメニュー側で閉じる）
  useEffect(() => {
    if (!editing || menuOpen) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !saving) setEditing(null);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [editing, menuOpen, saving]);

  // 地図をタップ → その位置に下書きピンを置いてメモ入力を開く。
  // シートを開いているときは、シートの外をタップしたことになるので閉じるだけにする
  const handleMapClick = (e: MapLayerMouseEvent) => {
    if (status !== "ready" || saving) return;
    if (editing) {
      setEditing(null);
      return;
    }
    if (!targetCategory) {
      setError("カテゴリがありません。右上のメニューからカテゴリを追加してください");
      return;
    }
    const { lat, lng } = e.lngLat;
    setError(null);
    setEditing({ pin: null, lat, lng, memo: "" });
  };

  const openPin = (pin: Pin) => {
    if (saving) return;
    setError(null);
    setEditing({ pin, lat: pin.lat, lng: pin.lng, memo: pin.memo });
  };

  const openMenu = () => {
    if (saving) return;
    setError(null);
    setEditing(null);
    setMenuOpen(true);
  };

  const handleSave = async (e: FormEvent) => {
    e.preventDefault();
    if (!editing || saving) return;
    setSaving(true);
    setError(null);
    try {
      const memo = editing.memo.trim();
      if (editing.pin) {
        const updated = await api.updatePin(editing.pin.categoryId, editing.pin.pinId, { memo });
        setPins((prev) => prev.map((p) => (p.pinId === updated.pinId ? updated : p)));
      } else {
        if (!targetCategory) return;
        const created = await api.createPin(targetCategory.categoryId, { lat: editing.lat, lng: editing.lng, memo });
        setPins((prev) => [...prev, created]);
      }
      setEditing(null);
    } catch (err) {
      setError(api.errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    const pin = editing?.pin;
    if (!pin || saving) return;
    setSaving(true);
    setError(null);
    try {
      await api.deletePin(pin.categoryId, pin.pinId);
      setPins((prev) => prev.filter((p) => p.pinId !== pin.pinId));
      setEditing(null);
    } catch (err) {
      setError(api.errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  // ドラッグで位置を直す。先に画面を動かし、保存に失敗したら元の位置に戻す
  const handleDragEnd = async (pin: Pin, lat: number, lng: number) => {
    setError(null);
    setPins((prev) => prev.map((p) => (p.pinId === pin.pinId ? { ...p, lat, lng } : p)));
    try {
      const updated = await api.updatePin(pin.categoryId, pin.pinId, { lat, lng });
      setPins((prev) => prev.map((p) => (p.pinId === updated.pinId ? updated : p)));
    } catch (err) {
      setPins((prev) => prev.map((p) => (p.pinId === pin.pinId ? pin : p)));
      setError(api.errorMessage(err));
    }
  };

  // ---- カテゴリ（メニューから） ----

  const handleToggleCategory = (categoryId: string) => {
    const next = hiddenSet.has(categoryId) ? hiddenIds.filter((id) => id !== categoryId) : [...hiddenIds, categoryId];
    setHiddenIds(next);
    saveHiddenCategoryIds(next);
  };

  const handleCreateCategory = async (name: string) => {
    setSaving(true);
    setError(null);
    try {
      const created = await api.createCategory({ name, color: nextCategoryColor(categories.length) });
      setCategories((prev) => [...prev, created]);
      return true;
    } catch (err) {
      setError(api.errorMessage(err));
      return false;
    } finally {
      setSaving(false);
    }
  };

  const handleRenameCategory = async (categoryId: string, name: string) => {
    setSaving(true);
    setError(null);
    try {
      const updated = await api.updateCategory(categoryId, { name });
      setCategories((prev) => prev.map((c) => (c.categoryId === updated.categoryId ? updated : c)));
      return true;
    } catch (err) {
      setError(api.errorMessage(err));
      return false;
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteCategory = async (category: Category) => {
    const ok = window.confirm(
      `「${category.name}」を削除しますか？\nこのカテゴリにだけ入っているピンも一緒に削除されます。`,
    );
    if (!ok) return;
    setSaving(true);
    setError(null);
    try {
      await api.deleteCategory(category.categoryId);
      setCategories((prev) => prev.filter((c) => c.categoryId !== category.categoryId));
      setPins((prev) => prev.filter((p) => p.categoryId !== category.categoryId));
      if (hiddenSet.has(category.categoryId)) {
        const next = hiddenIds.filter((id) => id !== category.categoryId);
        setHiddenIds(next);
        saveHiddenCategoryIds(next);
      }
    } catch (err) {
      setError(api.errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      className={styles.root}
      data-sheet-open={sheetOpen || undefined}
      style={{ "--sheet-height": `${sheetHeight}px` } as CSSProperties}
    >
      <Map
        // 前回の位置から開く。初回は環境変数の初期位置（東京駅）
        initialViewState={lastView ?? DEFAULT_VIEW}
        mapStyle={MAP_STYLE}
        style={{ width: "100%", height: "100%" }}
        onClick={handleMapClick}
        onLoad={() => {
          // 初回だけ自動で現在地へ移動する。2回目以降は右上のボタンを押したときだけ
          if (!lastView) geolocateRef.current?.trigger();
        }}
        onMoveEnd={(e) => saveLastView(e.viewState)}
      >
        <NavigationControl position="top-right" />
        <GeolocateControl
          ref={geolocateRef}
          position="top-right"
          positionOptions={{ enableHighAccuracy: true }}
          trackUserLocation
          showUserLocation
        />

        {visiblePins.map((pin) => (
          <Marker
            key={pin.pinId}
            latitude={pin.lat}
            longitude={pin.lng}
            anchor="bottom-left"
            draggable={!saving}
            onClick={(e) => {
              e.originalEvent.stopPropagation(); // 地図の onClick（新規ピン）を発火させない
              openPin(pin);
            }}
            onDragEnd={(e) => handleDragEnd(pin, e.lngLat.lat, e.lngLat.lng)}
          >
            <div
              className={styles.pin}
              style={{ "--pin-color": categoryById.get(pin.categoryId)?.color } as CSSProperties}
              title={pin.memo || "(メモなし)"}
            />
          </Marker>
        ))}

        {editing && !editing.pin && (
          <Marker latitude={editing.lat} longitude={editing.lng} anchor="bottom-left">
            <div
              className={`${styles.pin} ${styles.draft}`}
              style={{ "--pin-color": targetCategory?.color } as CSSProperties}
            />
          </Marker>
        )}
      </Map>

      <button type="button" className={styles.menuButton} onClick={openMenu} aria-label="メニューを開く">
        <span aria-hidden>☰</span>
      </button>

      {/* 読み込み中とエラーだけ、左上に出す（メニューを開いているときはメニューの中に出す） */}
      {!menuOpen && (status === "loading" || error) && (
        <div className={styles.status} role={error ? "alert" : "status"}>
          {error ?? "ピンを読み込んでいます…"}
        </div>
      )}

      {menuOpen && (
        <Menu
          categories={categories}
          hiddenIds={hiddenSet}
          busy={saving}
          error={error}
          onToggle={handleToggleCategory}
          onCreate={handleCreateCategory}
          onRename={handleRenameCategory}
          onDelete={handleDeleteCategory}
          onClose={() => setMenuOpen(false)}
        />
      )}

      {editing && (
        <form ref={sheetRef} className={styles.sheet} onSubmit={handleSave}>
          <div className={styles.sheetHeader}>
            <h2 className={styles.sheetTitle}>{editing.pin ? "メモを編集" : "メモを追加"}</h2>
            <button
              type="button"
              className={styles.sheetClose}
              onClick={() => setEditing(null)}
              disabled={saving}
              aria-label="閉じる"
            >
              ×
            </button>
          </div>
          <div className={styles.coord}>
            {editing.lat.toFixed(5)}, {editing.lng.toFixed(5)}
          </div>
          <textarea
            className={styles.memo}
            value={editing.memo}
            onChange={(e) => setEditing({ ...editing, memo: e.target.value })}
            placeholder="例：気になるカフェ。土日は混むらしい"
            maxLength={MEMO_MAX_LENGTH}
            autoFocus
          />
          <div className={styles.row}>
            {editing.pin && (
              <button
                type="button"
                className={`${styles.button} ${styles.danger}`}
                onClick={handleDelete}
                disabled={saving}
              >
                削除
              </button>
            )}
            <button type="submit" className={`${styles.button} ${styles.primary}`} disabled={saving}>
              {saving ? "保存中…" : "保存"}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
