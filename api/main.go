package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	categoriesTable := os.Getenv("CATEGORIES_TABLE")
	pinsTable := os.Getenv("PINS_TABLE")
	if categoriesTable == "" || pinsTable == "" {
		log.Fatal("環境変数 CATEGORIES_TABLE と PINS_TABLE を設定してください")
	}

	// 認証情報とリージョンは Lambda の実行ロールと環境から自動で読み込まれる
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	store := NewDynamoStore(dynamodb.NewFromConfig(awsCfg), categoriesTable, pinsTable)
	lambda.Start(NewApp(store, cfg).Handle)
}
