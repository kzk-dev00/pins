package main

import (
	"cmp"
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const (
	guestTTL      = 30 * 24 * time.Hour // ゲストのデータの期限
	renewWithin   = 7 * 24 * time.Hour  // 残りがこれを切ったら期限を延ばす
	defaultColor  = "#E0472F"
	sortOrderStep = 1000
	categoryIDLen = 10
	pinIDLen      = 12
	viewKeyLen    = 22
)

// Config は上限の設定。ピンの上限は1カテゴリあたりの数。
type Config struct {
	GuestMaxCategories       int
	GuestMaxPinsPerCategory  int
	MemberMaxCategories      int
	MemberMaxPinsPerCategory int
}

type App struct {
	store  Store
	cfg    Config
	now    func() time.Time
	routes []route
}

// request は振り分け後のリクエスト。params はパスの {categoryId} などの値。
type request struct {
	events.APIGatewayV2HTTPRequest
	params []string
	owner  owner
}

type handlerFunc func(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error)

type route struct {
	method  string
	pattern []string // "*" は任意の1区切り
	public  bool     // 持ち主の識別がいらない
	handle  handlerFunc
}

func NewApp(store Store, cfg Config) *App {
	a := &App{store: store, cfg: cfg, now: time.Now}
	a.routes = []route{
		{http.MethodGet, []string{"health"}, true, a.health},
		{http.MethodGet, []string{"categories"}, false, a.listCategories},
		{http.MethodPost, []string{"categories"}, false, a.createCategory},
		{http.MethodPatch, []string{"categories", "*"}, false, a.updateCategory},
		{http.MethodDelete, []string{"categories", "*"}, false, a.deleteCategory},
		{http.MethodPost, []string{"categories", "*", "share"}, false, a.shareCategory},
		{http.MethodDelete, []string{"categories", "*", "share"}, false, a.unshareCategory},
		{http.MethodGet, []string{"pins"}, false, a.listPins},
		{http.MethodPost, []string{"categories", "*", "pins"}, false, a.createPin},
		{http.MethodPatch, []string{"categories", "*", "pins", "*"}, false, a.updatePin},
		{http.MethodDelete, []string{"categories", "*", "pins", "*"}, false, a.deletePin},
		{http.MethodGet, []string{"s", "*"}, true, a.getShared},
	}
	return a
}

// Handle は Lambda の入口。エラーも含めて必ず HTTP のレスポンスを返す。
func (a *App) Handle(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	resp, err := a.dispatch(ctx, req)
	if err != nil {
		var ae *apiError
		if !errors.As(err, &ae) {
			log.Printf("ERROR %s %s: %v", req.RequestContext.HTTP.Method, requestPath(req), err)
			ae = errInternal
		}
		return errorResponse(ae), nil
	}
	return resp, nil
}

func (a *App) dispatch(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	// ANY /{proxy+} だと CORS のプリフライトもここに届く。CORS のヘッダーは API Gateway が付けるので、成功だけ返す
	if method == http.MethodOptions {
		return noContent()
	}
	segments := strings.Split(strings.Trim(requestPath(req), "/"), "/")
	pathMatched := false
	for _, rt := range a.routes {
		params, ok := matchPattern(rt.pattern, segments)
		if !ok {
			continue
		}
		pathMatched = true
		if rt.method != method {
			continue
		}
		r := &request{APIGatewayV2HTTPRequest: req, params: params}
		if !rt.public {
			o, err := identifyOwner(req)
			if err != nil {
				return events.APIGatewayV2HTTPResponse{}, err
			}
			r.owner = o
		}
		return rt.handle(ctx, r)
	}
	if pathMatched {
		return events.APIGatewayV2HTTPResponse{}, errMethodNotAllowed
	}
	return events.APIGatewayV2HTTPResponse{}, errRouteNotFound
}

// requestPath は ANY /{proxy+} の proxy からパスを作る（ステージ名を含まないようにするため）。
func requestPath(req events.APIGatewayV2HTTPRequest) string {
	if p, ok := req.PathParameters["proxy"]; ok {
		return "/" + p
	}
	return req.RawPath
}

func matchPattern(pattern, segments []string) ([]string, bool) {
	if len(pattern) != len(segments) {
		return nil, false
	}
	var params []string
	for i, p := range pattern {
		switch {
		case p == "*" && segments[i] != "":
			params = append(params, segments[i])
		case p != segments[i]:
			return nil, false
		}
	}
	return params, true
}

func (a *App) health(_ context.Context, _ *request) (events.APIGatewayV2HTTPResponse, error) {
	return jsonResponse(http.StatusOK, map[string]string{"status": "ok"})
}

// ---- カテゴリ ----

type categoryInput struct {
	Name      *string  `json:"name"`
	Color     *string  `json:"color"`
	SortOrder *float64 `json:"sortOrder"`
}

// validate は入力をチェックし、名前と色を保存する形に整える。
func (in *categoryInput) validate() error {
	if in.Name != nil {
		name, ok := normalizeCategoryName(*in.Name)
		if !ok {
			return invalidInput("カテゴリ名は1〜30文字で入力してください")
		}
		in.Name = &name
	}
	if in.Color != nil {
		color, ok := normalizeColor(*in.Color)
		if !ok {
			return invalidInput("色は #RRGGBB の形式で指定してください")
		}
		in.Color = &color
	}
	return nil
}

func (a *App) listCategories(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	cats, err := a.activeCategories(ctx, r.owner.id)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	for i := range cats {
		if err := a.renewAndSave(ctx, &cats[i]); err != nil {
			return events.APIGatewayV2HTTPResponse{}, err
		}
	}
	slices.SortFunc(cats, func(x, y Category) int {
		return cmp.Or(
			cmp.Compare(x.SortOrder, y.SortOrder),
			strings.Compare(x.CreatedAt, y.CreatedAt),
			strings.Compare(x.CategoryID, y.CategoryID),
		)
	})
	out := make([]categoryJSON, 0, len(cats))
	for _, c := range cats {
		out = append(out, c.toJSON())
	}
	return jsonResponse(http.StatusOK, map[string]any{"categories": out})
}

func (a *App) createCategory(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	var in categoryInput
	if err := decodeJSON(r.APIGatewayV2HTTPRequest, &in); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	if in.Name == nil {
		return events.APIGatewayV2HTTPResponse{}, invalidInput("カテゴリ名は1〜30文字で入力してください")
	}
	if err := in.validate(); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	cats, err := a.activeCategories(ctx, r.owner.id)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	maxCategories, _ := a.limits(r.owner)
	if len(cats) >= maxCategories {
		return events.APIGatewayV2HTTPResponse{}, categoryLimitError(maxCategories)
	}

	id, err := randomID(categoryIDLen)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	now := a.timestamp()
	c := Category{
		OwnerID:    r.owner.id,
		CategoryID: id,
		Name:       *in.Name,
		Color:      cmp.Or(deref(in.Color), defaultColor),
		SortOrder:  nextSortOrder(cats),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if in.SortOrder != nil {
		c.SortOrder = *in.SortOrder
	}
	if r.owner.guest {
		c.ExpiresAt = a.now().Add(guestTTL).Unix()
	}
	if err := a.store.CreateCategory(ctx, c); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return jsonResponse(http.StatusCreated, c.toJSON())
}

func (a *App) updateCategory(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	var in categoryInput
	if err := decodeJSON(r.APIGatewayV2HTTPRequest, &in); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	if err := in.validate(); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	c, err := a.loadCategory(ctx, r.owner.id, r.params[0])
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	renewed := a.renewExpiry(&c)
	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.Color != nil {
		c.Color = *in.Color
	}
	if in.SortOrder != nil {
		c.SortOrder = *in.SortOrder
	}
	c.UpdatedAt = a.timestamp()
	if err := a.saveCategory(ctx, c, renewed); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return jsonResponse(http.StatusOK, c.toJSON())
}

func (a *App) deleteCategory(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	c, err := a.loadCategory(ctx, r.owner.id, r.params[0])
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	// 途中で失敗してもピンだけが残らないように、ピンを先に消す
	pins, err := a.store.ListPins(ctx, c.OwnerID, c.CategoryID)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	for _, p := range pins {
		if err := a.store.DeletePin(ctx, p.OwnerID, p.CategoryID, p.PinID); err != nil && !errors.Is(err, ErrNotFound) {
			return events.APIGatewayV2HTTPResponse{}, err
		}
	}
	if err := a.store.DeleteCategory(ctx, c.OwnerID, c.CategoryID); err != nil && !errors.Is(err, ErrNotFound) {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return noContent()
}

func (a *App) shareCategory(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	c, err := a.loadCategory(ctx, r.owner.id, r.params[0])
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	viewKey, err := randomID(viewKeyLen)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	renewed := a.renewExpiry(&c)
	c.ViewKey = viewKey // 共有中なら作り直し。古いキーは使えなくなる
	c.UpdatedAt = a.timestamp()
	if err := a.saveCategory(ctx, c, renewed); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return jsonResponse(http.StatusOK, map[string]string{"viewKey": viewKey})
}

func (a *App) unshareCategory(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	c, err := a.loadCategory(ctx, r.owner.id, r.params[0])
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	if c.ViewKey == "" {
		return noContent()
	}
	renewed := a.renewExpiry(&c)
	c.ViewKey = ""
	c.UpdatedAt = a.timestamp()
	if err := a.saveCategory(ctx, c, renewed); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return noContent()
}

// ---- ピン ----

type pinInput struct {
	Lat  *float64 `json:"lat"`
	Lng  *float64 `json:"lng"`
	Memo *string  `json:"memo"`
}

func (in *pinInput) validate() error {
	if in.Lat != nil && !validLat(*in.Lat) {
		return invalidInput("緯度は -90〜90 の範囲で指定してください")
	}
	if in.Lng != nil && !validLng(*in.Lng) {
		return invalidInput("経度は -180〜180 の範囲で指定してください")
	}
	if in.Memo != nil && !validMemo(*in.Memo) {
		return invalidInput("メモは200文字以内で入力してください")
	}
	return nil
}

func (a *App) listPins(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	categoryID := r.QueryStringParameters["categoryId"]
	if categoryID != "" {
		if _, err := a.loadCategory(ctx, r.owner.id, categoryID); err != nil {
			return events.APIGatewayV2HTTPResponse{}, err
		}
	}
	pins, err := a.activePins(ctx, r.owner.id, categoryID)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	out := make([]pinJSON, 0, len(pins))
	for _, p := range pins {
		out = append(out, p.toJSON())
	}
	return jsonResponse(http.StatusOK, map[string]any{"pins": out})
}

func (a *App) createPin(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	var in pinInput
	if err := decodeJSON(r.APIGatewayV2HTTPRequest, &in); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	if in.Lat == nil || in.Lng == nil {
		return events.APIGatewayV2HTTPResponse{}, invalidInput("緯度と経度を指定してください")
	}
	if err := in.validate(); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	c, err := a.loadCategory(ctx, r.owner.id, r.params[0])
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	pins, err := a.activePins(ctx, c.OwnerID, c.CategoryID)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	_, maxPins := a.limits(r.owner)
	if len(pins) >= maxPins {
		return events.APIGatewayV2HTTPResponse{}, pinLimitError(maxPins)
	}
	if err := a.renewAndSave(ctx, &c); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	id, err := randomID(pinIDLen)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	now := a.timestamp()
	p := Pin{
		OwnerID:    c.OwnerID,
		SK:         pinSK(c.CategoryID, id),
		PinID:      id,
		CategoryID: c.CategoryID,
		Lat:        *in.Lat,
		Lng:        *in.Lng,
		Memo:       deref(in.Memo),
		CreatedAt:  now,
		UpdatedAt:  now,
		ExpiresAt:  c.ExpiresAt,
	}
	if err := a.store.CreatePin(ctx, p); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return jsonResponse(http.StatusCreated, p.toJSON())
}

func (a *App) updatePin(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	var in pinInput
	if err := decodeJSON(r.APIGatewayV2HTTPRequest, &in); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	if err := in.validate(); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	c, p, err := a.loadPin(ctx, r.owner.id, r.params[0], r.params[1])
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	if err := a.renewAndSave(ctx, &c); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	if in.Lat != nil {
		p.Lat = *in.Lat
	}
	if in.Lng != nil {
		p.Lng = *in.Lng
	}
	if in.Memo != nil {
		p.Memo = *in.Memo
	}
	p.ExpiresAt = c.ExpiresAt
	p.UpdatedAt = a.timestamp()
	if err := a.store.UpdatePin(ctx, p); err != nil {
		if errors.Is(err, ErrNotFound) {
			return events.APIGatewayV2HTTPResponse{}, errPinNotFound
		}
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return jsonResponse(http.StatusOK, p.toJSON())
}

func (a *App) deletePin(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	_, p, err := a.loadPin(ctx, r.owner.id, r.params[0], r.params[1])
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	if err := a.store.DeletePin(ctx, p.OwnerID, p.CategoryID, p.PinID); err != nil && !errors.Is(err, ErrNotFound) {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return noContent()
}

// ---- 共有 ----

func (a *App) getShared(ctx context.Context, r *request) (events.APIGatewayV2HTTPResponse, error) {
	viewKey := r.params[0]
	if !isID(viewKey, viewKeyLen) {
		return events.APIGatewayV2HTTPResponse{}, errSharedNotFound
	}
	found, err := a.store.FindCategoryByViewKey(ctx, viewKey)
	if errors.Is(err, ErrNotFound) {
		return events.APIGatewayV2HTTPResponse{}, errSharedNotFound
	}
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	// GSI は結果整合なので、共有をやめた直後の古い結果を返さないよう、本体のテーブルで確かめ直す
	c, err := a.store.GetCategory(ctx, found.OwnerID, found.CategoryID)
	if errors.Is(err, ErrNotFound) || (err == nil && (c.ViewKey != viewKey || isExpired(c.ExpiresAt, a.now()))) {
		return events.APIGatewayV2HTTPResponse{}, errSharedNotFound
	}
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	if err := a.renewAndSave(ctx, &c); err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	pins, err := a.activePins(ctx, c.OwnerID, c.CategoryID)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	out := make([]sharedPinJSON, 0, len(pins))
	for _, p := range pins {
		out = append(out, p.toSharedJSON())
	}
	return jsonResponse(http.StatusOK, map[string]any{
		"category": sharedCategoryJSON{Name: c.Name, Color: c.Color},
		"pins":     out,
	})
}

// ---- 共通の処理 ----

func (a *App) limits(o owner) (maxCategories, maxPinsPerCategory int) {
	if o.guest {
		return a.cfg.GuestMaxCategories, a.cfg.GuestMaxPinsPerCategory
	}
	return a.cfg.MemberMaxCategories, a.cfg.MemberMaxPinsPerCategory
}

func (a *App) timestamp() string {
	return a.now().UTC().Format(time.RFC3339)
}

// activeCategories は期限切れを除いたカテゴリを返す（DynamoDB は期限ぴったりには消さないため）。
func (a *App) activeCategories(ctx context.Context, ownerID string) ([]Category, error) {
	cats, err := a.store.ListCategories(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	now := a.now()
	return slices.DeleteFunc(cats, func(c Category) bool { return isExpired(c.ExpiresAt, now) }), nil
}

// activePins は期限切れを除いたピンを、作成順に返す。
func (a *App) activePins(ctx context.Context, ownerID, categoryID string) ([]Pin, error) {
	pins, err := a.store.ListPins(ctx, ownerID, categoryID)
	if err != nil {
		return nil, err
	}
	now := a.now()
	pins = slices.DeleteFunc(pins, func(p Pin) bool { return isExpired(p.ExpiresAt, now) })
	slices.SortFunc(pins, func(x, y Pin) int {
		return cmp.Or(strings.Compare(x.CreatedAt, y.CreatedAt), strings.Compare(x.PinID, y.PinID))
	})
	return pins, nil
}

// loadCategory は持ち主のカテゴリを返す。ないときと期限切れのときは 404 にする。
func (a *App) loadCategory(ctx context.Context, ownerID, categoryID string) (Category, error) {
	if !isID(categoryID, categoryIDLen) {
		return Category{}, errCategoryNotFound
	}
	c, err := a.store.GetCategory(ctx, ownerID, categoryID)
	if errors.Is(err, ErrNotFound) || (err == nil && isExpired(c.ExpiresAt, a.now())) {
		return Category{}, errCategoryNotFound
	}
	return c, err
}

// loadPin はカテゴリとピンを返す。どちらかがないときと期限切れのときは 404 にする。
func (a *App) loadPin(ctx context.Context, ownerID, categoryID, pinID string) (Category, Pin, error) {
	c, err := a.loadCategory(ctx, ownerID, categoryID)
	if err != nil {
		return Category{}, Pin{}, err
	}
	if !isID(pinID, pinIDLen) {
		return Category{}, Pin{}, errPinNotFound
	}
	p, err := a.store.GetPin(ctx, ownerID, categoryID, pinID)
	if errors.Is(err, ErrNotFound) || (err == nil && isExpired(p.ExpiresAt, a.now())) {
		return Category{}, Pin{}, errPinNotFound
	}
	return c, p, err
}

// renewExpiry はゲストのカテゴリの残りが7日を切っていれば、期限を30日後に延ばして true を返す。
// 毎回延ばすと書き込みが増えるので、残りが少ないときだけにしている。
func (a *App) renewExpiry(c *Category) bool {
	if c.ExpiresAt == 0 {
		return false
	}
	now := a.now()
	if time.Unix(c.ExpiresAt, 0).Sub(now) >= renewWithin {
		return false
	}
	c.ExpiresAt = now.Add(guestTTL).Unix()
	return true
}

// renewAndSave は必要なら期限を延ばし、カテゴリとピンに保存する。
func (a *App) renewAndSave(ctx context.Context, c *Category) error {
	if !a.renewExpiry(c) {
		return nil
	}
	return a.saveCategory(ctx, *c, true)
}

// saveCategory はカテゴリを保存する。期限を延ばしたときは、ピンの期限もそろえる。
func (a *App) saveCategory(ctx context.Context, c Category, renewed bool) error {
	if err := a.store.UpdateCategory(ctx, c); err != nil {
		if errors.Is(err, ErrNotFound) {
			return errCategoryNotFound
		}
		return err
	}
	if !renewed {
		return nil
	}
	pins, err := a.store.ListPins(ctx, c.OwnerID, c.CategoryID)
	if err != nil {
		return err
	}
	for _, p := range pins {
		if p.ExpiresAt == c.ExpiresAt {
			continue
		}
		p.ExpiresAt = c.ExpiresAt
		if err := a.store.UpdatePin(ctx, p); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

// nextSortOrder は最後尾の並び順（最大値＋1000）を返す。
func nextSortOrder(cats []Category) float64 {
	if len(cats) == 0 {
		return sortOrderStep
	}
	maxOrder := cats[0].SortOrder
	for _, c := range cats[1:] {
		maxOrder = max(maxOrder, c.SortOrder)
	}
	return maxOrder + sortOrderStep
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
