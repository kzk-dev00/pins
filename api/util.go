package main

import (
	"crypto/rand"
	"regexp"
	"strings"
	"unicode/utf8"
)

const idAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// randomID は推測できないランダムな英数字を n 文字作る。
func randomID(n int) (string, error) {
	out := make([]byte, 0, n)
	buf := make([]byte, n*2)
	for len(out) < n {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			// 248 = 62 × 4。これ以上の値を捨てて、文字の出やすさに偏りが出ないようにする
			if b >= 248 {
				continue
			}
			out = append(out, idAlphabet[int(b)%len(idAlphabet)])
			if len(out) == n {
				break
			}
		}
	}
	return string(out), nil
}

// isID は s が randomID(n) で作れる形かどうかを返す。
func isID(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(idAlphabet, rune(s[i])) {
			return false
		}
	}
	return true
}

var colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

const (
	maxCategoryNameLen = 30
	maxPinTitleLen     = 20
	maxMemoLen         = 200
)

// normalizeText は前後の空白を消し、min〜max文字かを確かめる。
func normalizeText(text string, min, max int) (string, bool) {
	text = strings.TrimSpace(text)
	n := utf8.RuneCountInString(text)
	return text, n >= min && n <= max
}

// normalizeColor は #RRGGBB の形かを確かめ、大文字にそろえる。
func normalizeColor(color string) (string, bool) {
	if !colorPattern.MatchString(color) {
		return "", false
	}
	return strings.ToUpper(color), true
}

// validMemo はメモが0〜200文字かを確かめる（空白は消さない）。
func validMemo(memo string) bool {
	return utf8.RuneCountInString(memo) <= maxMemoLen
}

// validCoord は緯度経度が -limit〜limit の範囲かを確かめる（緯度は90、経度は180を渡す）。
func validCoord(v, limit float64) bool {
	return v >= -limit && v <= limit
}
