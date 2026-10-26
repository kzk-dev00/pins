package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const day = 24 * time.Hour

var (
	tokenA = strings.Repeat("a", 40)
	tokenB = strings.Repeat("b", 40)
)

var testConfig = Config{
	GuestMaxCategories:       3,
	GuestMaxPinsPerCategory:  10,
	MemberMaxCategories:      50,
	MemberMaxPinsPerCategory: 1000,
}

// testEnv はメモリ上の保存先と、自由に進められる時計を持つテスト環境。
type testEnv struct {
	t     *testing.T
	app   *App
	store *MemoryStore
	now   time.Time
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	e := &testEnv{t: t, store: NewMemoryStore(), now: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)}
	e.app = NewApp(e.store, testConfig)
	e.app.now = func() time.Time { return e.now }
	return e
}

func (e *testEnv) advance(d time.Duration) {
	e.now = e.now.Add(d)
}

// auth はリクエストに持ち主の情報を付ける。
type auth func(*events.APIGatewayV2HTTPRequest)

func guest(token string) auth {
	return func(r *events.APIGatewayV2HTTPRequest) { r.Headers["x-guest-token"] = token }
}

func member(sub string) auth {
	return func(r *events.APIGatewayV2HTTPRequest) {
		r.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
			JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{"sub": sub}},
		}
	}
}

func anonymous(*events.APIGatewayV2HTTPRequest) {}

type testResponse struct {
	t      *testing.T
	status int
	body   string
}

// do は API Gateway（HTTP API, ANY /{proxy+}）から届くのと同じ形のリクエストを作って処理させる。
func (e *testEnv) do(as auth, method, target, body string) testResponse {
	e.t.Helper()
	u, err := url.Parse(target)
	if err != nil {
		e.t.Fatal(err)
	}
	req := events.APIGatewayV2HTTPRequest{
		RawPath:        u.Path,
		RawQueryString: u.RawQuery,
		Headers:        map[string]string{"content-type": "application/json"},
		PathParameters: map[string]string{"proxy": strings.TrimPrefix(u.Path, "/")},
		Body:           body,
	}
	if q := u.Query(); len(q) > 0 {
		req.QueryStringParameters = map[string]string{}
		for k := range q {
			req.QueryStringParameters[k] = q.Get(k)
		}
	}
	req.RequestContext.HTTP.Method = method
	as(&req)

	resp, err := e.app.Handle(context.Background(), req)
	if err != nil {
		e.t.Fatalf("Handle がエラーを返した: %v", err)
	}
	return testResponse{t: e.t, status: resp.StatusCode, body: resp.Body}
}

func (r testResponse) expect(status int) testResponse {
	r.t.Helper()
	if r.status != status {
		r.t.Fatalf("status = %d, want %d（body: %s）", r.status, status, r.body)
	}
	return r
}

func (r testResponse) decode(v any) {
	r.t.Helper()
	if err := json.Unmarshal([]byte(r.body), v); err != nil {
		r.t.Fatalf("JSON を読めない: %v（body: %s）", err, r.body)
	}
}

func (r testResponse) expectError(status int, code string) {
	r.t.Helper()
	r.expect(status)
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	r.decode(&body)
	if body.Error.Code != code {
		r.t.Fatalf("error.code = %q, want %q", body.Error.Code, code)
	}
	if body.Error.Message == "" {
		r.t.Fatal("error.message が空")
	}
}

func (e *testEnv) createCategory(as auth, body string) categoryJSON {
	e.t.Helper()
	var c categoryJSON
	e.do(as, http.MethodPost, "/categories", body).expect(http.StatusCreated).decode(&c)
	return c
}

// createPin はピンを作る。categoryIDs に入れるカテゴリを指定する
func (e *testEnv) createPin(as auth, lat, lng float64, memo string, categoryIDs ...string) pinJSON {
	e.t.Helper()
	body, _ := json.Marshal(map[string]any{"lat": lat, "lng": lng, "memo": memo, "categoryIds": categoryIDs})
	var p pinJSON
	e.do(as, http.MethodPost, "/pins", string(body)).expect(http.StatusCreated).decode(&p)
	return p
}

func (e *testEnv) listCategories(as auth) []categoryJSON {
	e.t.Helper()
	var out struct {
		Categories []categoryJSON `json:"categories"`
	}
	e.do(as, http.MethodGet, "/categories", "").expect(http.StatusOK).decode(&out)
	return out.Categories
}

func (e *testEnv) listPins(as auth) []pinJSON {
	e.t.Helper()
	var out struct {
		Pins []pinJSON `json:"pins"`
	}
	e.do(as, http.MethodGet, "/pins", "").expect(http.StatusOK).decode(&out)
	return out.Pins
}

func (e *testEnv) share(as auth, categoryID string) string {
	e.t.Helper()
	var out struct {
		ViewKey string `json:"viewKey"`
	}
	e.do(as, http.MethodPost, "/categories/"+categoryID+"/share", "").expect(http.StatusOK).decode(&out)
	if !isID(out.ViewKey, viewKeyLen) {
		e.t.Fatalf("viewKey = %q", out.ViewKey)
	}
	return out.ViewKey
}

func pinByID(pins []pinJSON, pinID string) (pinJSON, bool) {
	for _, p := range pins {
		if p.PinID == pinID {
			return p, true
		}
	}
	return pinJSON{}, false
}

func TestHealth(t *testing.T) {
	e := newTestEnv(t)
	e.do(anonymous, http.MethodGet, "/health", "").expect(http.StatusOK)
}

func TestRouting(t *testing.T) {
	e := newTestEnv(t)
	e.do(guest(tokenA), http.MethodGet, "/unknown", "").expectError(http.StatusNotFound, "not_found")
	e.do(guest(tokenA), http.MethodPut, "/categories", "").expectError(http.StatusMethodNotAllowed, "method_not_allowed")
	// 古いパスはもう使えない
	e.do(anonymous, http.MethodGet, "/s/"+strings.Repeat("a", viewKeyLen), "").expectError(http.StatusNotFound, "not_found")
	e.do(guest(tokenA), http.MethodPost, "/categories/ABCDEFGHIJ/pins", `{}`).expectError(http.StatusNotFound, "not_found")
}

func TestPreflight(t *testing.T) {
	e := newTestEnv(t)
	// ブラウザのプリフライトにはゲストトークンが付かないので、トークンなしでも成功する
	e.do(anonymous, http.MethodOptions, "/categories", "").expect(http.StatusNoContent)
	e.do(anonymous, http.MethodOptions, "/pins/ABCDEFGHIJKL", "").expect(http.StatusNoContent)
}

func TestGuestToken(t *testing.T) {
	e := newTestEnv(t)
	e.do(anonymous, http.MethodGet, "/categories", "").expectError(http.StatusUnauthorized, "unauthorized")
	e.do(guest(strings.Repeat("a", 31)), http.MethodGet, "/categories", "").expectError(http.StatusUnauthorized, "unauthorized")
	e.do(guest(strings.Repeat("a", 129)), http.MethodGet, "/categories", "").expectError(http.StatusUnauthorized, "unauthorized")
	e.do(guest(strings.Repeat("a", 31)+"!"), http.MethodGet, "/categories", "").expectError(http.StatusUnauthorized, "unauthorized")
	e.do(guest(strings.Repeat("a", 32)), http.MethodGet, "/categories", "").expect(http.StatusOK)
	e.do(guest(strings.Repeat("A-_9", 32)), http.MethodGet, "/categories", "").expect(http.StatusOK)
}

func TestGuestTokenIsHashed(t *testing.T) {
	e := newTestEnv(t)
	e.createCategory(guest(tokenA), `{"name":"厚木ランチ"}`)
	cats, _ := e.store.ListCategories(context.Background(), guestOwnerID(tokenA))
	if len(cats) != 1 {
		t.Fatalf("ハッシュした ownerId で保存されていない: %d 件", len(cats))
	}
	if strings.Contains(cats[0].OwnerID, tokenA) {
		t.Fatal("トークンそのものが保存されている")
	}
}

func TestCategoryLifecycle(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)

	c1 := e.createCategory(as, `{"name":"  厚木ランチ  "}`)
	if c1.Name != "厚木ランチ" {
		t.Errorf("name = %q（前後の空白が消えていない）", c1.Name)
	}
	if c1.Color != defaultColor {
		t.Errorf("color = %q, want %q", c1.Color, defaultColor)
	}
	if c1.SortOrder != 1000 {
		t.Errorf("sortOrder = %v, want 1000", c1.SortOrder)
	}
	if !isID(c1.CategoryID, categoryIDLen) {
		t.Errorf("categoryId = %q", c1.CategoryID)
	}
	if want := e.now.Add(30 * day).Unix(); c1.ExpiresAt != want {
		t.Errorf("expiresAt = %d, want %d（30日後）", c1.ExpiresAt, want)
	}

	c2 := e.createCategory(as, `{"name":"旅行候補","color":"#1a2b3c"}`)
	if c2.SortOrder != 2000 {
		t.Errorf("sortOrder = %v, want 2000（最大値＋1000）", c2.SortOrder)
	}
	if c2.Color != "#1A2B3C" {
		t.Errorf("color = %q（大文字にそろっていない）", c2.Color)
	}

	// 中間の値を送って先頭に移す
	e.do(as, http.MethodPatch, "/categories/"+c2.CategoryID, `{"sortOrder":500}`).expect(http.StatusOK)
	if cats := e.listCategories(as); len(cats) != 2 || cats[0].CategoryID != c2.CategoryID {
		t.Fatalf("並び順が sortOrder 順になっていない: %+v", cats)
	}

	var updated categoryJSON
	e.do(as, http.MethodPatch, "/categories/"+c1.CategoryID, `{"name":"新しい名前"}`).expect(http.StatusOK).decode(&updated)
	if updated.Name != "新しい名前" || updated.Color != defaultColor {
		t.Errorf("編集結果が違う: %+v", updated)
	}

	e.do(as, http.MethodDelete, "/categories/"+c1.CategoryID, "").expect(http.StatusNoContent)
	e.do(as, http.MethodDelete, "/categories/"+c1.CategoryID, "").expectError(http.StatusNotFound, "category_not_found")
	if cats := e.listCategories(as); len(cats) != 1 {
		t.Fatalf("削除後の件数 = %d, want 1", len(cats))
	}
}

func TestCategoryValidation(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"名前なし", `{}`, http.StatusBadRequest, "invalid_input"},
		{"空白だけ", `{"name":"　 "}`, http.StatusBadRequest, "invalid_input"},
		{"31文字", `{"name":"` + strings.Repeat("あ", 31) + `"}`, http.StatusBadRequest, "invalid_input"},
		{"色の形式違い", `{"name":"a","color":"red"}`, http.StatusBadRequest, "invalid_input"},
		{"知らない項目", `{"name":"a","extra":1}`, http.StatusBadRequest, "invalid_json"},
		{"JSON ではない", `not json`, http.StatusBadRequest, "invalid_json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv(t)
			e.do(guest(tokenA), http.MethodPost, "/categories", tc.body).expectError(tc.status, tc.code)
		})
	}

	t.Run("30文字はOK", func(t *testing.T) {
		e := newTestEnv(t)
		e.createCategory(guest(tokenA), `{"name":"`+strings.Repeat("あ", 30)+`"}`)
	})
}

func TestCategoryLimitIgnoresExpired(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	for range testConfig.GuestMaxCategories {
		e.createCategory(as, `{"name":"a"}`)
	}
	e.do(as, http.MethodPost, "/categories", `{"name":"a"}`).expectError(http.StatusConflict, "category_limit")

	// 期限が切れたカテゴリは DynamoDB に残っていても数えない
	e.advance(30*day + time.Second)
	e.createCategory(as, `{"name":"b"}`)
	if cats := e.listCategories(as); len(cats) != 1 {
		t.Fatalf("件数 = %d, want 1", len(cats))
	}
}

func TestOwnersAreIsolated(t *testing.T) {
	e := newTestEnv(t)
	c := e.createCategory(guest(tokenA), `{"name":"A のカテゴリ"}`)
	p := e.createPin(guest(tokenA), 35, 139, "", c.CategoryID)

	b := guest(tokenB)
	if cats := e.listCategories(b); len(cats) != 0 {
		t.Fatalf("B に A のカテゴリが見えている: %+v", cats)
	}
	if pins := e.listPins(b); len(pins) != 0 {
		t.Fatalf("B に A のピンが見えている: %+v", pins)
	}
	e.do(b, http.MethodPatch, "/categories/"+c.CategoryID, `{"name":"x"}`).expectError(http.StatusNotFound, "category_not_found")
	e.do(b, http.MethodDelete, "/categories/"+c.CategoryID, "").expectError(http.StatusNotFound, "category_not_found")
	e.do(b, http.MethodPost, "/categories/"+c.CategoryID+"/share", "").expectError(http.StatusNotFound, "category_not_found")
	// A のカテゴリに B のピンは入れられない
	e.do(b, http.MethodPost, "/pins", `{"lat":1,"lng":1,"categoryIds":["`+c.CategoryID+`"]}`).expectError(http.StatusNotFound, "category_not_found")
	e.do(b, http.MethodPatch, "/pins/"+p.PinID, `{"memo":"x"}`).expectError(http.StatusNotFound, "pin_not_found")
	e.do(b, http.MethodDelete, "/pins/"+p.PinID, "").expectError(http.StatusNotFound, "pin_not_found")
}

func TestMemberHasNoExpiry(t *testing.T) {
	e := newTestEnv(t)
	as := member("cognito-sub-1")
	c := e.createCategory(as, `{"name":"会員のカテゴリ"}`)
	if c.ExpiresAt != 0 {
		t.Fatalf("会員のカテゴリに期限が入っている: %d", c.ExpiresAt)
	}
	p := e.createPin(as, 35, 139, "", c.CategoryID)
	if p.ExpiresAt != 0 {
		t.Fatalf("会員のピンに期限が入っている: %d", p.ExpiresAt)
	}
	if _, err := e.store.GetCategory(context.Background(), "u_cognito-sub-1", c.CategoryID); err != nil {
		t.Fatalf("ownerId が u_ + sub になっていない: %v", err)
	}

	e.advance(365 * day)
	if cats := e.listCategories(as); len(cats) != 1 {
		t.Fatal("会員のカテゴリが期限切れ扱いになっている")
	}
	if pins := e.listPins(as); len(pins) != 1 {
		t.Fatal("会員のピンが期限切れ扱いになっている")
	}
}

func TestPinLifecycle(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	lunch := e.createCategory(as, `{"name":"厚木ランチ"}`)
	trip := e.createCategory(as, `{"name":"旅行候補"}`)

	// 1つのピンを2つのカテゴリに入れる
	p := e.createPin(as, 35.4430, 139.3625, "駅前のそば屋", lunch.CategoryID, trip.CategoryID)
	if !isID(p.PinID, pinIDLen) || p.Memo != "駅前のそば屋" {
		t.Fatalf("作成結果が違う: %+v", p)
	}
	if len(p.CategoryIDs) != 2 || p.CategoryIDs[0] != lunch.CategoryID || p.CategoryIDs[1] != trip.CategoryID {
		t.Fatalf("categoryIds = %v", p.CategoryIDs)
	}
	e.createPin(as, 34, 135, "", trip.CategoryID)

	if pins := e.listPins(as); len(pins) != 2 {
		t.Fatalf("全部のピン = %d 件, want 2", len(pins))
	}

	var updated pinJSON
	path := "/pins/" + p.PinID
	e.do(as, http.MethodPatch, path, `{"memo":"閉店していた"}`).expect(http.StatusOK).decode(&updated)
	if updated.Memo != "閉店していた" || updated.Lat != p.Lat || len(updated.CategoryIDs) != 2 {
		t.Fatalf("メモだけ変えたはずが違う: %+v", updated)
	}

	// カテゴリの付け替え
	e.do(as, http.MethodPatch, path, `{"categoryIds":["`+trip.CategoryID+`"]}`).expect(http.StatusOK).decode(&updated)
	if len(updated.CategoryIDs) != 1 || updated.CategoryIDs[0] != trip.CategoryID {
		t.Fatalf("カテゴリを付け替えられていない: %v", updated.CategoryIDs)
	}

	e.do(as, http.MethodPatch, path, `{"lat":36,"lng":140}`).expect(http.StatusOK).decode(&updated)
	if updated.Lat != 36 || updated.Lng != 140 {
		t.Fatalf("位置が変わっていない: %+v", updated)
	}

	e.do(as, http.MethodDelete, path, "").expect(http.StatusNoContent)
	e.do(as, http.MethodDelete, path, "").expectError(http.StatusNotFound, "pin_not_found")
	e.do(as, http.MethodPatch, path, `{"memo":"x"}`).expectError(http.StatusNotFound, "pin_not_found")
}

func TestPinValidation(t *testing.T) {
	cases := []struct {
		name   string
		body   func(categoryID string) string
		status int
		code   string
	}{
		{"緯度なし", func(c string) string { return `{"lng":139,"categoryIds":["` + c + `"]}` }, http.StatusBadRequest, "invalid_input"},
		{"経度なし", func(c string) string { return `{"lat":35,"categoryIds":["` + c + `"]}` }, http.StatusBadRequest, "invalid_input"},
		{"緯度が範囲外", func(c string) string { return `{"lat":90.1,"lng":139,"categoryIds":["` + c + `"]}` }, http.StatusBadRequest, "invalid_input"},
		{"経度が範囲外", func(c string) string { return `{"lat":35,"lng":-180.1,"categoryIds":["` + c + `"]}` }, http.StatusBadRequest, "invalid_input"},
		{"メモが201文字", func(c string) string {
			return `{"lat":35,"lng":139,"memo":"` + strings.Repeat("あ", 201) + `","categoryIds":["` + c + `"]}`
		}, http.StatusBadRequest, "invalid_input"},
		{"緯度が文字列", func(c string) string { return `{"lat":"35","lng":139,"categoryIds":["` + c + `"]}` }, http.StatusBadRequest, "invalid_json"},
		{"カテゴリなし", func(string) string { return `{"lat":35,"lng":139}` }, http.StatusBadRequest, "invalid_input"},
		{"カテゴリが空", func(string) string { return `{"lat":35,"lng":139,"categoryIds":[]}` }, http.StatusBadRequest, "invalid_input"},
		{"カテゴリが重複", func(c string) string { return `{"lat":35,"lng":139,"categoryIds":["` + c + `","` + c + `"]}` }, http.StatusBadRequest, "invalid_input"},
		{"ないカテゴリ", func(string) string { return `{"lat":35,"lng":139,"categoryIds":["ABCDEFGHIJ"]}` }, http.StatusNotFound, "category_not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv(t)
			c := e.createCategory(guest(tokenA), `{"name":"a"}`)
			e.do(guest(tokenA), http.MethodPost, "/pins", tc.body(c.CategoryID)).expectError(tc.status, tc.code)
		})
	}

	t.Run("境界値はOK", func(t *testing.T) {
		e := newTestEnv(t)
		c := e.createCategory(guest(tokenA), `{"name":"a"}`)
		e.createPin(guest(tokenA), -90, 180, strings.Repeat("あ", 200), c.CategoryID)
	})

	t.Run("編集でカテゴリを空にはできない", func(t *testing.T) {
		e := newTestEnv(t)
		c := e.createCategory(guest(tokenA), `{"name":"a"}`)
		p := e.createPin(guest(tokenA), 35, 139, "", c.CategoryID)
		e.do(guest(tokenA), http.MethodPatch, "/pins/"+p.PinID, `{"categoryIds":[]}`).expectError(http.StatusBadRequest, "invalid_input")
	})
}

func TestPinLimitPerCategory(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	a := e.createCategory(as, `{"name":"a"}`)
	b := e.createCategory(as, `{"name":"b"}`)
	var pins []pinJSON
	for range testConfig.GuestMaxPinsPerCategory {
		pins = append(pins, e.createPin(as, 35, 139, "", a.CategoryID))
	}
	e.do(as, http.MethodPost, "/pins", `{"lat":35,"lng":139,"categoryIds":["`+a.CategoryID+`"]}`).expectError(http.StatusConflict, "pin_limit")
	// どれか1つのカテゴリでも上限なら作れない
	e.do(as, http.MethodPost, "/pins", `{"lat":35,"lng":139,"categoryIds":["`+b.CategoryID+`","`+a.CategoryID+`"]}`).expectError(http.StatusConflict, "pin_limit")

	// 上限は1カテゴリあたりなので、別のカテゴリには追加できる
	other := e.createPin(as, 35, 139, "", b.CategoryID)
	// 満員のカテゴリへ付け替えることはできない
	e.do(as, http.MethodPatch, "/pins/"+other.PinID, `{"categoryIds":["`+a.CategoryID+`"]}`).expectError(http.StatusConflict, "pin_limit")
	// すでに入っているカテゴリは数え直さないので、満員の a に入っているピンに b を足すのは OK
	e.do(as, http.MethodPatch, "/pins/"+pins[0].PinID, `{"categoryIds":["`+a.CategoryID+`","`+b.CategoryID+`"]}`).expect(http.StatusOK)
}

func TestPinExpiryFollowsLatestCategory(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	early := e.createCategory(as, `{"name":"先に作った"}`)
	e.advance(10 * day)
	late := e.createCategory(as, `{"name":"あとで作った"}`)

	p := e.createPin(as, 35, 139, "", early.CategoryID, late.CategoryID)
	if p.ExpiresAt != late.ExpiresAt {
		t.Fatalf("ピンの期限 %d が、一番遅いカテゴリの期限 %d になっていない", p.ExpiresAt, late.ExpiresAt)
	}

	// 先に作ったカテゴリが期限切れになっても、あとのカテゴリに入っているのでピンは残る
	e.advance(25 * day)
	if pins := e.listPins(as); len(pins) != 1 {
		t.Fatalf("ピンが消えている: %+v", pins)
	}
}

func TestGuestExpiryIsRenewedWhenNearlyExpired(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	e.createPin(as, 35, 139, "", c.CategoryID)

	// 残り10日：まだ延ばさない
	e.advance(20 * day)
	if cats := e.listCategories(as); cats[0].ExpiresAt != c.ExpiresAt {
		t.Fatalf("残り10日なのに期限が変わった: %d → %d", c.ExpiresAt, cats[0].ExpiresAt)
	}

	// 残り6日：カテゴリ一覧を開いたら30日後に延びる
	e.advance(4 * day)
	want := e.now.Add(30 * day).Unix()
	if cats := e.listCategories(as); cats[0].ExpiresAt != want {
		t.Fatalf("期限 = %d, want %d", cats[0].ExpiresAt, want)
	}
	if pins := e.listPins(as); pins[0].ExpiresAt != want {
		t.Fatalf("ピンの期限がカテゴリとそろっていない: %d, want %d", pins[0].ExpiresAt, want)
	}
}

func TestGuestExpiryIsRenewedByPinAndShare(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	viewKey := e.share(as, c.CategoryID)

	// ピンの追加で延びる
	e.advance(25 * day)
	p := e.createPin(as, 35, 139, "", c.CategoryID)
	if want := e.now.Add(30 * day).Unix(); p.ExpiresAt != want {
		t.Fatalf("ピン追加で期限が延びていない: %d, want %d", p.ExpiresAt, want)
	}

	// 共有URLを開かれて延びる
	e.advance(25 * day)
	e.do(anonymous, http.MethodGet, "/share/"+viewKey, "").expect(http.StatusOK)
	want := e.now.Add(30 * day).Unix()
	got, _ := e.store.GetCategory(context.Background(), guestOwnerID(tokenA), c.CategoryID)
	if got.ExpiresAt != want {
		t.Fatalf("共有URLで期限が延びていない: %d, want %d", got.ExpiresAt, want)
	}
	if pins := e.listPins(as); pins[0].ExpiresAt != want {
		t.Fatalf("ピンの期限がそろっていない: %d, want %d", pins[0].ExpiresAt, want)
	}
}

func TestExpiredDataIsHidden(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	p := e.createPin(as, 35, 139, "", c.CategoryID)

	e.advance(30*day + time.Second)
	if cats := e.listCategories(as); len(cats) != 0 {
		t.Fatalf("期限切れのカテゴリが返っている: %+v", cats)
	}
	if pins := e.listPins(as); len(pins) != 0 {
		t.Fatalf("期限切れのピンが返っている: %+v", pins)
	}
	e.do(as, http.MethodPatch, "/categories/"+c.CategoryID, `{"name":"x"}`).expectError(http.StatusNotFound, "category_not_found")
	e.do(as, http.MethodPatch, "/pins/"+p.PinID, `{"memo":"x"}`).expectError(http.StatusNotFound, "pin_not_found")
	e.do(as, http.MethodPost, "/pins", `{"lat":1,"lng":1,"categoryIds":["`+c.CategoryID+`"]}`).expectError(http.StatusNotFound, "category_not_found")
}

func TestShareLifecycle(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	lunch := e.createCategory(as, `{"name":"厚木ランチ","color":"#00AA00"}`)
	secret := e.createCategory(as, `{"name":"自宅メモ"}`)
	onlyLunch := e.createPin(as, 35.44, 139.36, "そば", lunch.CategoryID)
	both := e.createPin(as, 35.45, 139.37, "両方", lunch.CategoryID, secret.CategoryID)
	e.createPin(as, 35.46, 139.38, "見せたくない", secret.CategoryID)

	key1 := e.share(as, lunch.CategoryID)
	resp := e.do(anonymous, http.MethodGet, "/share/"+key1, "").expect(http.StatusOK)
	for _, leak := range []string{"ownerId", "g_", "expiresAt", "categoryIds", secret.CategoryID, "自宅メモ", "見せたくない"} {
		if strings.Contains(resp.body, leak) {
			t.Fatalf("共有ページに %q が含まれている: %s", leak, resp.body)
		}
	}
	var shared struct {
		Category sharedCategoryJSON `json:"category"`
		Pins     []sharedPinJSON    `json:"pins"`
	}
	resp.decode(&shared)
	if shared.Category.Name != "厚木ランチ" || shared.Category.Color != "#00AA00" || len(shared.Pins) != 2 {
		t.Fatalf("共有ページの中身が違う: %+v", shared)
	}
	got := map[string]bool{shared.Pins[0].PinID: true, shared.Pins[1].PinID: true}
	if !got[onlyLunch.PinID] || !got[both.PinID] {
		t.Fatalf("共有したカテゴリのピンだけになっていない: %+v", shared.Pins)
	}

	// 作り直すと古いキーは使えなくなる
	key2 := e.share(as, lunch.CategoryID)
	if key2 == key1 {
		t.Fatal("viewKey が作り直されていない")
	}
	e.do(anonymous, http.MethodGet, "/share/"+key1, "").expectError(http.StatusNotFound, "shared_not_found")
	e.do(anonymous, http.MethodGet, "/share/"+key2, "").expect(http.StatusOK)

	// 共有をやめる
	e.do(as, http.MethodDelete, "/categories/"+lunch.CategoryID+"/share", "").expect(http.StatusNoContent)
	e.do(as, http.MethodDelete, "/categories/"+lunch.CategoryID+"/share", "").expect(http.StatusNoContent)
	e.do(anonymous, http.MethodGet, "/share/"+key2, "").expectError(http.StatusNotFound, "shared_not_found")

	e.do(anonymous, http.MethodGet, "/share/short", "").expectError(http.StatusNotFound, "shared_not_found")
}

func TestSharedPageHidesExpiredCategory(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	viewKey := e.share(as, c.CategoryID)

	e.advance(30*day + time.Second)
	e.do(anonymous, http.MethodGet, "/share/"+viewKey, "").expectError(http.StatusNotFound, "shared_not_found")
}

func TestDeleteCategory(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	a := e.createCategory(as, `{"name":"a"}`)
	e.advance(5 * day)
	b := e.createCategory(as, `{"name":"b"}`)
	onlyA := e.createPin(as, 35, 139, "", a.CategoryID)
	both := e.createPin(as, 36, 140, "", a.CategoryID, b.CategoryID)
	onlyB := e.createPin(as, 37, 141, "", b.CategoryID)

	e.do(as, http.MethodDelete, "/categories/"+b.CategoryID, "").expect(http.StatusNoContent)

	pins := e.listPins(as)
	if _, ok := pinByID(pins, onlyB.PinID); ok {
		t.Fatal("どのカテゴリにも入らなくなったピンが残っている")
	}
	if _, ok := pinByID(pins, onlyA.PinID); !ok {
		t.Fatal("削除したカテゴリに入っていないピンが消えた")
	}
	got, ok := pinByID(pins, both.PinID)
	if !ok {
		t.Fatal("ほかのカテゴリにも入っているピンが消えた")
	}
	if len(got.CategoryIDs) != 1 || got.CategoryIDs[0] != a.CategoryID {
		t.Fatalf("削除したカテゴリが外れていない: %v", got.CategoryIDs)
	}
	// 残ったカテゴリ a の期限に合わせて計算し直される
	if got.ExpiresAt != a.ExpiresAt {
		t.Fatalf("ピンの期限 %d が、残ったカテゴリの期限 %d になっていない", got.ExpiresAt, a.ExpiresAt)
	}
}

func TestOldFormatPinsAreHidden(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	// 複数カテゴリ対応より前の形（categoryIds がない）のピン
	old := Pin{OwnerID: guestOwnerID(tokenA), SK: "ABCDEFGHIJ#ABCDEFGHIJKL", PinID: "ABCDEFGHIJKL"}
	if err := e.store.CreatePin(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	if pins := e.listPins(as); len(pins) != 0 {
		t.Fatalf("古い形のピンが返っている: %+v", pins)
	}
}

func TestLoadConfig(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	cfg, err := loadConfig(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg != (Config{3, 10, 50, 1000}) {
		t.Fatalf("初期値が違う: %+v", cfg)
	}

	cfg, err = loadConfig(env(map[string]string{"GUEST_MAX_PINS": "5", "MEMBER_MAX_CATEGORIES": "100"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GuestMaxPinsPerCategory != 5 || cfg.MemberMaxCategories != 100 || cfg.GuestMaxCategories != 3 {
		t.Fatalf("環境変数が反映されていない: %+v", cfg)
	}

	for _, v := range []string{"0", "-1", "abc", "1.5"} {
		if _, err := loadConfig(env(map[string]string{"GUEST_MAX_CATEGORIES": v})); err == nil {
			t.Errorf("%q でエラーにならない", v)
		}
	}
}

func TestRandomID(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		id, err := randomID(viewKeyLen)
		if err != nil {
			t.Fatal(err)
		}
		if !isID(id, viewKeyLen) {
			t.Fatalf("形式が違う: %q", id)
		}
		if seen[id] {
			t.Fatalf("重複した: %q", id)
		}
		seen[id] = true
	}
}

func TestPinTitle(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	body := func(title string) string {
		return `{"lat":35,"lng":139,"title":"` + title + `","categoryIds":["` + c.CategoryID + `"]}`
	}

	// 前後の空白は消して保存する
	var p pinJSON
	e.do(as, http.MethodPost, "/pins", body("  カレー屋さん  ")).expect(http.StatusCreated).decode(&p)
	if p.Title != "カレー屋さん" {
		t.Fatalf("title = %q", p.Title)
	}

	// 20文字まで。21文字はエラー
	e.do(as, http.MethodPost, "/pins", body(strings.Repeat("あ", 20))).expect(http.StatusCreated)
	e.do(as, http.MethodPost, "/pins", body(strings.Repeat("あ", 21))).expectError(http.StatusBadRequest, "invalid_input")

	// タイトルは任意
	untitled := e.createPin(as, 35, 139, "", c.CategoryID)
	if untitled.Title != "" {
		t.Fatalf("title = %q, want 空", untitled.Title)
	}

	// 編集。送らなければ変わらず、空文字を送れば消える
	var updated pinJSON
	e.do(as, http.MethodPatch, "/pins/"+p.PinID, `{"memo":"辛口"}`).expect(http.StatusOK).decode(&updated)
	if updated.Title != "カレー屋さん" {
		t.Fatalf("タイトルを送っていないのに変わった: %q", updated.Title)
	}
	e.do(as, http.MethodPatch, "/pins/"+p.PinID, `{"title":"スープカレー"}`).expect(http.StatusOK).decode(&updated)
	if updated.Title != "スープカレー" || updated.Memo != "辛口" {
		t.Fatalf("編集結果が違う: %+v", updated)
	}
	e.do(as, http.MethodPatch, "/pins/"+p.PinID, `{"title":""}`).expect(http.StatusOK).decode(&updated)
	if updated.Title != "" {
		t.Fatalf("タイトルを消せていない: %q", updated.Title)
	}
	e.do(as, http.MethodPatch, "/pins/"+p.PinID, `{"title":"`+strings.Repeat("あ", 21)+`"}`).expectError(http.StatusBadRequest, "invalid_input")

	// 共有ページにもタイトルを返す
	e.do(as, http.MethodPatch, "/pins/"+p.PinID, `{"title":"カレー屋さん"}`).expect(http.StatusOK)
	viewKey := e.share(as, c.CategoryID)
	resp := e.do(anonymous, http.MethodGet, "/share/"+viewKey, "").expect(http.StatusOK)
	if !strings.Contains(resp.body, `"title":"カレー屋さん"`) {
		t.Fatalf("共有ページにタイトルがない: %s", resp.body)
	}
}

func TestPinWithoutTitleAttribute(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	// タイトルを追加する前に作ったピン（title の項目がない）も、空のタイトルとして返す
	old := Pin{OwnerID: guestOwnerID(tokenA), SK: "ABCDEFGHIJKL", PinID: "ABCDEFGHIJKL", CategoryIDs: []string{c.CategoryID}, ExpiresAt: c.ExpiresAt}
	if err := e.store.CreatePin(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	pins := e.listPins(as)
	if len(pins) != 1 || pins[0].Title != "" {
		t.Fatalf("古いピンが返らない、またはタイトルが空でない: %+v", pins)
	}
}
