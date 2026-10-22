package main

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// owner はリクエストした人。id はテーブルの ownerId。
type owner struct {
	id    string
	guest bool
}

var guestTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,128}$`)

// identifyOwner は持ち主を決める。
// 会員は API Gateway の JWT オーソライザーが検証した sub、ゲストは X-Guest-Token のハッシュを使う。
func identifyOwner(req events.APIGatewayV2HTTPRequest) (owner, error) {
	if auth := req.RequestContext.Authorizer; auth != nil && auth.JWT != nil {
		if sub := auth.JWT.Claims["sub"]; sub != "" {
			return owner{id: "u_" + sub}, nil
		}
	}
	token := header(req.Headers, "x-guest-token")
	if token == "" {
		return owner{}, errMissingToken
	}
	if !guestTokenPattern.MatchString(token) {
		return owner{}, errInvalidToken
	}
	return owner{id: guestOwnerID(token), guest: true}, nil
}

// guestOwnerID はトークンそのものではなく、ハッシュを DB に保存するための ID を作る。
func guestOwnerID(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "g_" + hex.EncodeToString(sum[:])
}

// header は大文字小文字を区別せずにヘッダーを取り出す（HTTP API は小文字で渡してくる）。
func header(headers map[string]string, name string) string {
	if v, ok := headers[name]; ok {
		return v
	}
	for k, v := range headers {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}
