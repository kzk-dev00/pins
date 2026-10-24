package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// apiError は画面にそのまま出せる日本語のメッセージを持つエラー。
type apiError struct {
	status  int
	code    string
	message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%d %s: %s", e.status, e.code, e.message)
}

var (
	errRouteNotFound    = &apiError{http.StatusNotFound, "not_found", "ページが見つかりません"}
	errMethodNotAllowed = &apiError{http.StatusMethodNotAllowed, "method_not_allowed", "この操作には対応していません"}
	errInvalidJSON      = &apiError{http.StatusBadRequest, "invalid_json", "リクエストの形式が正しくありません"}
	errMissingToken     = &apiError{http.StatusUnauthorized, "unauthorized", "ゲストトークンがありません。ページを再読み込みしてください"}
	errInvalidToken     = &apiError{http.StatusUnauthorized, "unauthorized", "ゲストトークンの形式が正しくありません"}
	errCategoryNotFound = &apiError{http.StatusNotFound, "category_not_found", "カテゴリが見つかりません"}
	errPinNotFound      = &apiError{http.StatusNotFound, "pin_not_found", "ピンが見つかりません"}
	errSharedNotFound   = &apiError{http.StatusNotFound, "shared_not_found", "共有されたカテゴリが見つかりません。共有が停止されたか、期限が切れた可能性があります"}
	errInternal         = &apiError{http.StatusInternalServerError, "internal", "サーバーでエラーが発生しました。時間をおいてもう一度お試しください"}
)

func invalidInput(message string) *apiError {
	return &apiError{http.StatusBadRequest, "invalid_input", message}
}

func categoryLimitError(max int) *apiError {
	return &apiError{http.StatusConflict, "category_limit", fmt.Sprintf("カテゴリは%d件まで作成できます", max)}
}

func pinLimitError(categoryName string, max int) *apiError {
	return &apiError{http.StatusConflict, "pin_limit", fmt.Sprintf("「%s」に登録できるピンは%d件までです", categoryName, max)}
}

func jsonHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/json; charset=utf-8"}
}

func jsonResponse(status int, v any) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: status, Headers: jsonHeaders(), Body: string(body)}, nil
}

func noContent() (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusNoContent}, nil
}

func errorResponse(e *apiError) events.APIGatewayV2HTTPResponse {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]string{"code": e.code, "message": e.message},
	})
	return events.APIGatewayV2HTTPResponse{StatusCode: e.status, Headers: jsonHeaders(), Body: string(body)}
}

// decodeJSON はリクエストの本文を v に読み込む。知らない項目があればエラーにする。
func decodeJSON(req events.APIGatewayV2HTTPRequest, v any) error {
	body := req.Body
	if req.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return errInvalidJSON
		}
		body = string(b)
	}
	dec := json.NewDecoder(strings.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errInvalidJSON
	}
	if dec.More() {
		return errInvalidJSON
	}
	return nil
}
