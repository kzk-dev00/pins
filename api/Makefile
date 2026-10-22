.PHONY: tidy test vet build zip clean

# go.mod と go.sum をコードに合わせて整える
tidy:
	go mod tidy

test:
	go test ./...

vet:
	go vet ./...

# Lambda（provided.al2023 / arm64）用の実行ファイル bootstrap を作る
build:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -trimpath -ldflags="-s -w" -o bootstrap .

# Lambda にアップロードする zip を作る
zip: build
	rm -f function.zip
	zip function.zip bootstrap

clean:
	rm -f bootstrap function.zip
