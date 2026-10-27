package main

import (
	"context"
	"errors"
)

var (
	// ErrNotFound は対象の項目がないことを表す。
	ErrNotFound = errors.New("not found")
	// ErrConflict は同じキーの項目がすでにあることを表す。
	ErrConflict = errors.New("already exists")
)

// Store はデータの保存先。本番は DynamoStore、テストは MemoryStore を使う。
// Create は同じキーがあれば ErrConflict、Update と Delete はキーがなければ ErrNotFound を返す。
// 期限切れのデータもそのまま返すので、絞り込みは呼び出し側で行う。
type Store interface {
	ListCategories(ctx context.Context, ownerID string) ([]Category, error)
	GetCategory(ctx context.Context, ownerID, categoryID string) (Category, error)
	CreateCategory(ctx context.Context, c Category) error
	UpdateCategory(ctx context.Context, c Category) error
	DeleteCategory(ctx context.Context, ownerID, categoryID string) error
	// GSI（viewKey-index）で引く。GSIは結果整合なので、共有の解除直後は古い結果が返ることがある
	FindCategoryByViewKey(ctx context.Context, viewKey string) (Category, error)

	// 持ち主のピンを全部返す（カテゴリでの絞り込みは呼び出し側で行う）
	ListPins(ctx context.Context, ownerID string) ([]Pin, error)
	GetPin(ctx context.Context, ownerID, pinID string) (Pin, error)
	CreatePin(ctx context.Context, p Pin) error
	UpdatePin(ctx context.Context, p Pin) error
	DeletePin(ctx context.Context, ownerID, pinID string) error
}
