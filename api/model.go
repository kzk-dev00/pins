package main

import (
	"slices"
	"time"
)

// Category は categories テーブルの1項目。
type Category struct {
	OwnerID    string  `dynamodbav:"ownerId"`
	CategoryID string  `dynamodbav:"categoryId"`
	Name       string  `dynamodbav:"name"`
	Color      string  `dynamodbav:"color"`
	SortOrder  float64 `dynamodbav:"sortOrder"`
	// 共有中のみ。空のときは項目ごと保存しない（空文字は GSI のキーにできない）
	ViewKey   string `dynamodbav:"viewKey,omitempty"`
	CreatedAt string `dynamodbav:"createdAt"`
	UpdatedAt string `dynamodbav:"updatedAt"`
	// ゲストのみ。秒単位の Unix 時刻（Number 型）。会員は 0 で、項目ごと保存しない
	ExpiresAt int64 `dynamodbav:"expiresAt,omitempty"`
}

// Pin は pins テーブルの1項目。1つのピンを複数のカテゴリに入れられる。
type Pin struct {
	OwnerID     string   `dynamodbav:"ownerId"`
	SK          string   `dynamodbav:"sk"` // pinId
	PinID       string   `dynamodbav:"pinId"`
	CategoryIDs []string `dynamodbav:"categoryIds"` // 1件以上
	Lat         float64  `dynamodbav:"lat"`
	Lng         float64  `dynamodbav:"lng"`
	Memo        string   `dynamodbav:"memo"`
	CreatedAt   string   `dynamodbav:"createdAt"`
	UpdatedAt   string   `dynamodbav:"updatedAt"`
	// 入っているカテゴリの期限のうち、一番遅いものにそろえる
	ExpiresAt int64 `dynamodbav:"expiresAt,omitempty"`
}

func (p Pin) hasCategory(categoryID string) bool {
	return slices.Contains(p.CategoryIDs, categoryID)
}

func isExpired(expiresAt int64, now time.Time) bool {
	return expiresAt != 0 && expiresAt <= now.Unix()
}

// 以下は API で返す JSON の形。ownerId は返さない。

type categoryJSON struct {
	CategoryID string  `json:"categoryId"`
	Name       string  `json:"name"`
	Color      string  `json:"color"`
	SortOrder  float64 `json:"sortOrder"`
	ViewKey    string  `json:"viewKey,omitempty"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
	ExpiresAt  int64   `json:"expiresAt,omitempty"`
}

type pinJSON struct {
	PinID       string   `json:"pinId"`
	CategoryIDs []string `json:"categoryIds"`
	Lat         float64  `json:"lat"`
	Lng         float64  `json:"lng"`
	Memo        string   `json:"memo"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	ExpiresAt   int64    `json:"expiresAt,omitempty"`
}

// 共有ページ用。持ち主に関わる情報（ownerId、期限、ほかのカテゴリ）は含めない。
type sharedCategoryJSON struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type sharedPinJSON struct {
	PinID     string  `json:"pinId"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Memo      string  `json:"memo"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

func (c Category) toJSON() categoryJSON {
	return categoryJSON{
		CategoryID: c.CategoryID,
		Name:       c.Name,
		Color:      c.Color,
		SortOrder:  c.SortOrder,
		ViewKey:    c.ViewKey,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
		ExpiresAt:  c.ExpiresAt,
	}
}

func (p Pin) toJSON() pinJSON {
	return pinJSON{
		PinID:       p.PinID,
		CategoryIDs: p.CategoryIDs,
		Lat:         p.Lat,
		Lng:         p.Lng,
		Memo:        p.Memo,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		ExpiresAt:   p.ExpiresAt,
	}
}

func (p Pin) toSharedJSON() sharedPinJSON {
	return sharedPinJSON{
		PinID:     p.PinID,
		Lat:       p.Lat,
		Lng:       p.Lng,
		Memo:      p.Memo,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
