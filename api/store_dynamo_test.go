package main

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDB に保存する形を確かめる（AWS には繋げない）。
func TestCategoryItemShape(t *testing.T) {
	item, err := attributevalue.MarshalMap(Category{OwnerID: "g_x", CategoryID: "c", Name: "n", Color: "#000000", SortOrder: 1000})
	if err != nil {
		t.Fatal(err)
	}
	// 空文字は GSI のキーにできないので、未共有なら viewKey は項目ごと保存しない
	if _, ok := item["viewKey"]; ok {
		t.Error("未共有なのに viewKey が保存されている")
	}
	if _, ok := item["expiresAt"]; ok {
		t.Error("会員なのに expiresAt が保存されている")
	}

	item, err = attributevalue.MarshalMap(Category{OwnerID: "g_x", CategoryID: "c", ViewKey: "k", ExpiresAt: 1790000000})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := item["viewKey"].(*types.AttributeValueMemberS); !ok {
		t.Error("viewKey が文字列型で保存されていない")
	}
	// TTL は Number 型でないと無視されて消えない
	if n, ok := item["expiresAt"].(*types.AttributeValueMemberN); !ok || n.Value != "1790000000" {
		t.Errorf("expiresAt が秒単位の Number 型になっていない: %#v", item["expiresAt"])
	}
}

func TestPinItemShape(t *testing.T) {
	item, err := attributevalue.MarshalMap(Pin{OwnerID: "g_x", SK: "pin", PinID: "pin", CategoryIDs: []string{"cat1", "cat2"}, ExpiresAt: 1790000000})
	if err != nil {
		t.Fatal(err)
	}
	if sk, ok := item["sk"].(*types.AttributeValueMemberS); !ok || sk.Value != "pin" {
		t.Errorf("sk が pinId になっていない: %#v", item["sk"])
	}
	// categoryIds は文字列の List で保存する
	list, ok := item["categoryIds"].(*types.AttributeValueMemberL)
	if !ok || len(list.Value) != 2 {
		t.Fatalf("categoryIds が List になっていない: %#v", item["categoryIds"])
	}
	if s, ok := list.Value[1].(*types.AttributeValueMemberS); !ok || s.Value != "cat2" {
		t.Errorf("categoryIds の中身が文字列になっていない: %#v", list.Value[1])
	}
	if _, ok := item["expiresAt"].(*types.AttributeValueMemberN); !ok {
		t.Error("expiresAt が Number 型になっていない")
	}
	if _, ok := item["title"].(*types.AttributeValueMemberS); !ok {
		t.Error("空のタイトルが文字列として保存されていない")
	}
	if _, ok := item["memo"].(*types.AttributeValueMemberS); !ok {
		t.Error("空のメモが文字列として保存されていない")
	}
}
