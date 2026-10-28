package main

import (
	"fmt"
	"strconv"
)

// loadConfig は環境変数から上限を読む。設定がなければ初期値を使う。
func loadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{
		GuestMaxCategories:       5,
		GuestMaxPinsPerCategory:  10,
		MemberMaxCategories:      50,
		MemberMaxPinsPerCategory: 1000,
	}
	fields := []struct {
		name string
		dst  *int
	}{
		{"GUEST_MAX_CATEGORIES", &cfg.GuestMaxCategories},
		{"GUEST_MAX_PINS", &cfg.GuestMaxPinsPerCategory},
		{"MEMBER_MAX_CATEGORIES", &cfg.MemberMaxCategories},
		{"MEMBER_MAX_PINS", &cfg.MemberMaxPinsPerCategory},
	}
	for _, f := range fields {
		v := getenv(f.name)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("環境変数%sは1以上の整数にしてください（今の値：%q）", f.name, v)
		}
		*f.dst = n
	}
	return cfg, nil
}
