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

func (e *testEnv) createPin(as auth, categoryID, body string) pinJSON {
	e.t.Helper()
	var p pinJSON
	e.do(as, http.MethodPost, "/categories/"+categoryID+"/pins", body).expect(http.StatusCreated).decode(&p)
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

func (e *testEnv) listPins(as auth, query string) []pinJSON {
	e.t.Helper()
	var out struct {
		Pins []pinJSON `json:"pins"`
	}
	e.do(as, http.MethodGet, "/pins"+query, "").expect(http.StatusOK).decode(&out)
	return out.Pins
}

func TestHealth(t *testing.T) {
	e := newTestEnv(t)
	e.do(anonymous, http.MethodGet, "/health", "").expect(http.StatusOK)
}

func TestRouting(t *testing.T) {
	e := newTestEnv(t)
	e.do(guest(tokenA), http.MethodGet, "/unknown", "").expectError(http.StatusNotFound, "not_found")
	e.do(guest(tokenA), http.MethodPut, "/categories", "").expectError(http.StatusMethodNotAllowed, "method_not_allowed")
}

func TestPreflight(t *testing.T) {
	e := newTestEnv(t)
	// ブラウザのプリフライトにはゲストトークンが付かないので、トークンなしでも成功する
	e.do(anonymous, http.MethodOptions, "/categories", "").expect(http.StatusNoContent)
	e.do(anonymous, http.MethodOptions, "/categories/ABCDEFGHIJ/pins", "").expect(http.StatusNoContent)
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
	p := e.createPin(guest(tokenA), c.CategoryID, `{"lat":35,"lng":139}`)

	b := guest(tokenB)
	if cats := e.listCategories(b); len(cats) != 0 {
		t.Fatalf("B に A のカテゴリが見えている: %+v", cats)
	}
	if pins := e.listPins(b, ""); len(pins) != 0 {
		t.Fatalf("B に A のピンが見えている: %+v", pins)
	}
	e.do(b, http.MethodPatch, "/categories/"+c.CategoryID, `{"name":"x"}`).expectError(http.StatusNotFound, "category_not_found")
	e.do(b, http.MethodDelete, "/categories/"+c.CategoryID, "").expectError(http.StatusNotFound, "category_not_found")
	e.do(b, http.MethodPost, "/categories/"+c.CategoryID+"/share", "").expectError(http.StatusNotFound, "category_not_found")
	e.do(b, http.MethodPost, "/categories/"+c.CategoryID+"/pins", `{"lat":1,"lng":1}`).expectError(http.StatusNotFound, "category_not_found")
	e.do(b, http.MethodDelete, "/categories/"+c.CategoryID+"/pins/"+p.PinID, "").expectError(http.StatusNotFound, "category_not_found")
}

func TestMemberHasNoExpiry(t *testing.T) {
	e := newTestEnv(t)
	as := member("cognito-sub-1")
	c := e.createCategory(as, `{"name":"会員のカテゴリ"}`)
	if c.ExpiresAt != 0 {
		t.Fatalf("会員のカテゴリに期限が入っている: %d", c.ExpiresAt)
	}
	p := e.createPin(as, c.CategoryID, `{"lat":35,"lng":139}`)
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
}

func TestPinLifecycle(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"厚木ランチ"}`)
	other := e.createCategory(as, `{"name":"旅行候補"}`)

	p := e.createPin(as, c.CategoryID, `{"lat":35.4430,"lng":139.3625,"memo":"駅前のそば屋"}`)
	if !isID(p.PinID, pinIDLen) || p.CategoryID != c.CategoryID || p.Memo != "駅前のそば屋" {
		t.Fatalf("作成結果が違う: %+v", p)
	}
	if p.ExpiresAt != c.ExpiresAt {
		t.Errorf("ピンの期限 %d がカテゴリの期限 %d とそろっていない", p.ExpiresAt, c.ExpiresAt)
	}
	e.createPin(as, other.CategoryID, `{"lat":34,"lng":135}`)

	if pins := e.listPins(as, ""); len(pins) != 2 {
		t.Fatalf("全部のピン = %d 件, want 2", len(pins))
	}
	if pins := e.listPins(as, "?categoryId="+c.CategoryID); len(pins) != 1 || pins[0].PinID != p.PinID {
		t.Fatalf("カテゴリで絞り込めていない: %+v", pins)
	}
	e.do(as, http.MethodGet, "/pins?categoryId=ABCDEFGHIJ", "").expectError(http.StatusNotFound, "category_not_found")

	var updated pinJSON
	path := "/categories/" + c.CategoryID + "/pins/" + p.PinID
	e.do(as, http.MethodPatch, path, `{"memo":"閉店していた"}`).expect(http.StatusOK).decode(&updated)
	if updated.Memo != "閉店していた" || updated.Lat != p.Lat || updated.Lng != p.Lng {
		t.Fatalf("編集結果が違う: %+v", updated)
	}
	e.do(as, http.MethodPatch, path, `{"memo":""}`).expect(http.StatusOK).decode(&updated)
	if updated.Memo != "" {
		t.Fatalf("メモを空にできていない: %q", updated.Memo)
	}

	e.do(as, http.MethodDelete, path, "").expect(http.StatusNoContent)
	e.do(as, http.MethodDelete, path, "").expectError(http.StatusNotFound, "pin_not_found")
	e.do(as, http.MethodPatch, path, `{"memo":"x"}`).expectError(http.StatusNotFound, "pin_not_found")
	e.do(as, http.MethodPost, "/categories/ABCDEFGHIJ/pins", `{"lat":1,"lng":1}`).expectError(http.StatusNotFound, "category_not_found")
}

func TestPinValidation(t *testing.T) {
	cases := []struct {
		name string
		body string
		code string
	}{
		{"緯度なし", `{"lng":139}`, "invalid_input"},
		{"経度なし", `{"lat":35}`, "invalid_input"},
		{"緯度が範囲外", `{"lat":90.1,"lng":139}`, "invalid_input"},
		{"経度が範囲外", `{"lat":35,"lng":-180.1}`, "invalid_input"},
		{"メモが201文字", `{"lat":35,"lng":139,"memo":"` + strings.Repeat("あ", 201) + `"}`, "invalid_input"},
		{"緯度が文字列", `{"lat":"35","lng":139}`, "invalid_json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv(t)
			c := e.createCategory(guest(tokenA), `{"name":"a"}`)
			e.do(guest(tokenA), http.MethodPost, "/categories/"+c.CategoryID+"/pins", tc.body).expectError(http.StatusBadRequest, tc.code)
		})
	}

	t.Run("境界値はOK", func(t *testing.T) {
		e := newTestEnv(t)
		c := e.createCategory(guest(tokenA), `{"name":"a"}`)
		e.createPin(guest(tokenA), c.CategoryID, `{"lat":-90,"lng":180,"memo":"`+strings.Repeat("あ", 200)+`"}`)
	})
}

func TestPinLimitPerCategory(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	for range testConfig.GuestMaxPinsPerCategory {
		e.createPin(as, c.CategoryID, `{"lat":35,"lng":139}`)
	}
	e.do(as, http.MethodPost, "/categories/"+c.CategoryID+"/pins", `{"lat":35,"lng":139}`).expectError(http.StatusConflict, "pin_limit")

	// 上限は1カテゴリあたりなので、別のカテゴリには追加できる
	other := e.createCategory(as, `{"name":"b"}`)
	e.createPin(as, other.CategoryID, `{"lat":35,"lng":139}`)
}

func TestGuestExpiryIsRenewedWhenNearlyExpired(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	e.createPin(as, c.CategoryID, `{"lat":35,"lng":139}`)

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
	if pins := e.listPins(as, ""); pins[0].ExpiresAt != want {
		t.Fatalf("ピンの期限がカテゴリとそろっていない: %d, want %d", pins[0].ExpiresAt, want)
	}
}

func TestGuestExpiryIsRenewedByPinAndShare(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	var share struct {
		ViewKey string `json:"viewKey"`
	}
	e.do(as, http.MethodPost, "/categories/"+c.CategoryID+"/share", "").expect(http.StatusOK).decode(&share)

	// ピンの追加で延びる
	e.advance(25 * day)
	p := e.createPin(as, c.CategoryID, `{"lat":35,"lng":139}`)
	if want := e.now.Add(30 * day).Unix(); p.ExpiresAt != want {
		t.Fatalf("ピン追加で期限が延びていない: %d, want %d", p.ExpiresAt, want)
	}

	// 共有URLを開かれて延びる
	e.advance(25 * day)
	e.do(anonymous, http.MethodGet, "/s/"+share.ViewKey, "").expect(http.StatusOK)
	want := e.now.Add(30 * day).Unix()
	got, _ := e.store.GetCategory(context.Background(), guestOwnerID(tokenA), c.CategoryID)
	if got.ExpiresAt != want {
		t.Fatalf("共有URLで期限が延びていない: %d, want %d", got.ExpiresAt, want)
	}
	if pins := e.listPins(as, ""); pins[0].ExpiresAt != want {
		t.Fatalf("ピンの期限がそろっていない: %d, want %d", pins[0].ExpiresAt, want)
	}
}

func TestExpiredDataIsHidden(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	p := e.createPin(as, c.CategoryID, `{"lat":35,"lng":139}`)

	e.advance(30*day + time.Second)
	if cats := e.listCategories(as); len(cats) != 0 {
		t.Fatalf("期限切れのカテゴリが返っている: %+v", cats)
	}
	if pins := e.listPins(as, ""); len(pins) != 0 {
		t.Fatalf("期限切れのピンが返っている: %+v", pins)
	}
	e.do(as, http.MethodPatch, "/categories/"+c.CategoryID, `{"name":"x"}`).expectError(http.StatusNotFound, "category_not_found")
	e.do(as, http.MethodPatch, "/categories/"+c.CategoryID+"/pins/"+p.PinID, `{"memo":"x"}`).expectError(http.StatusNotFound, "category_not_found")
}

func TestShareLifecycle(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"厚木ランチ","color":"#00AA00"}`)
	e.createPin(as, c.CategoryID, `{"lat":35.44,"lng":139.36,"memo":"そば"}`)

	share := func() string {
		t.Helper()
		var out struct {
			ViewKey string `json:"viewKey"`
		}
		e.do(as, http.MethodPost, "/categories/"+c.CategoryID+"/share", "").expect(http.StatusOK).decode(&out)
		if !isID(out.ViewKey, viewKeyLen) {
			t.Fatalf("viewKey = %q", out.ViewKey)
		}
		return out.ViewKey
	}

	key1 := share()
	resp := e.do(anonymous, http.MethodGet, "/s/"+key1, "").expect(http.StatusOK)
	if strings.Contains(resp.body, "ownerId") || strings.Contains(resp.body, "g_") {
		t.Fatalf("共有ページに持ち主の情報が含まれている: %s", resp.body)
	}
	var shared struct {
		Category sharedCategoryJSON `json:"category"`
		Pins     []sharedPinJSON    `json:"pins"`
	}
	resp.decode(&shared)
	if shared.Category.Name != "厚木ランチ" || shared.Category.Color != "#00AA00" || len(shared.Pins) != 1 || shared.Pins[0].Memo != "そば" {
		t.Fatalf("共有ページの中身が違う: %+v", shared)
	}
	if cats := e.listCategories(as); cats[0].ViewKey != key1 {
		t.Fatalf("カテゴリ一覧に viewKey が出ていない: %+v", cats[0])
	}

	// 作り直すと古いキーは使えなくなる
	key2 := share()
	if key2 == key1 {
		t.Fatal("viewKey が作り直されていない")
	}
	e.do(anonymous, http.MethodGet, "/s/"+key1, "").expectError(http.StatusNotFound, "shared_not_found")
	e.do(anonymous, http.MethodGet, "/s/"+key2, "").expect(http.StatusOK)

	// 共有をやめる
	e.do(as, http.MethodDelete, "/categories/"+c.CategoryID+"/share", "").expect(http.StatusNoContent)
	e.do(as, http.MethodDelete, "/categories/"+c.CategoryID+"/share", "").expect(http.StatusNoContent)
	e.do(anonymous, http.MethodGet, "/s/"+key2, "").expectError(http.StatusNotFound, "shared_not_found")
	if cats := e.listCategories(as); cats[0].ViewKey != "" {
		t.Fatalf("共有をやめても viewKey が残っている: %+v", cats[0])
	}

	e.do(anonymous, http.MethodGet, "/s/short", "").expectError(http.StatusNotFound, "shared_not_found")
}

func TestSharedPageHidesExpiredCategory(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	var out struct {
		ViewKey string `json:"viewKey"`
	}
	e.do(as, http.MethodPost, "/categories/"+c.CategoryID+"/share", "").expect(http.StatusOK).decode(&out)

	e.advance(30*day + time.Second)
	e.do(anonymous, http.MethodGet, "/s/"+out.ViewKey, "").expectError(http.StatusNotFound, "shared_not_found")
}

func TestDeleteCategoryDeletesPins(t *testing.T) {
	e := newTestEnv(t)
	as := guest(tokenA)
	c := e.createCategory(as, `{"name":"a"}`)
	keep := e.createCategory(as, `{"name":"b"}`)
	e.createPin(as, c.CategoryID, `{"lat":35,"lng":139}`)
	e.createPin(as, c.CategoryID, `{"lat":36,"lng":140}`)
	e.createPin(as, keep.CategoryID, `{"lat":37,"lng":141}`)

	e.do(as, http.MethodDelete, "/categories/"+c.CategoryID, "").expect(http.StatusNoContent)

	pins, _ := e.store.ListPins(context.Background(), guestOwnerID(tokenA), "")
	if len(pins) != 1 || pins[0].CategoryID != keep.CategoryID {
		t.Fatalf("削除したカテゴリのピンが残っている、または別のカテゴリのピンが消えた: %+v", pins)
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
