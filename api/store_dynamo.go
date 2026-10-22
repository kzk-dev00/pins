package main

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const viewKeyIndex = "viewKey-index"

// DynamoStore は DynamoDB への読み書き。
type DynamoStore struct {
	client          *dynamodb.Client
	categoriesTable string
	pinsTable       string
}

func NewDynamoStore(client *dynamodb.Client, categoriesTable, pinsTable string) *DynamoStore {
	return &DynamoStore{client: client, categoriesTable: categoriesTable, pinsTable: pinsTable}
}

func attrS(v string) types.AttributeValue {
	return &types.AttributeValueMemberS{Value: v}
}

func categoryKey(ownerID, categoryID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"ownerId": attrS(ownerID), "categoryId": attrS(categoryID)}
}

func pinKey(ownerID, categoryID, pinID string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"ownerId": attrS(ownerID), "sk": attrS(pinSK(categoryID, pinID))}
}

func (s *DynamoStore) ListCategories(ctx context.Context, ownerID string) ([]Category, error) {
	return queryAll[Category](ctx, s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(s.categoriesTable),
		KeyConditionExpression:    aws.String("ownerId = :o"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":o": attrS(ownerID)},
	})
}

func (s *DynamoStore) GetCategory(ctx context.Context, ownerID, categoryID string) (Category, error) {
	return getItem[Category](ctx, s.client, s.categoriesTable, categoryKey(ownerID, categoryID))
}

func (s *DynamoStore) CreateCategory(ctx context.Context, c Category) error {
	return s.put(ctx, s.categoriesTable, c, "attribute_not_exists(ownerId)", ErrConflict)
}

func (s *DynamoStore) UpdateCategory(ctx context.Context, c Category) error {
	// 項目ごと置き換えるので、ViewKey を空にすると viewKey 属性が消えて GSI からも外れる
	return s.put(ctx, s.categoriesTable, c, "attribute_exists(ownerId)", ErrNotFound)
}

func (s *DynamoStore) DeleteCategory(ctx context.Context, ownerID, categoryID string) error {
	return s.delete(ctx, s.categoriesTable, categoryKey(ownerID, categoryID))
}

func (s *DynamoStore) FindCategoryByViewKey(ctx context.Context, viewKey string) (Category, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(s.categoriesTable),
		IndexName:                 aws.String(viewKeyIndex),
		KeyConditionExpression:    aws.String("viewKey = :v"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":v": attrS(viewKey)},
		Limit:                     aws.Int32(1),
	})
	if err != nil {
		return Category{}, err
	}
	if len(out.Items) == 0 {
		return Category{}, ErrNotFound
	}
	var c Category
	err = attributevalue.UnmarshalMap(out.Items[0], &c)
	return c, err
}

func (s *DynamoStore) ListPins(ctx context.Context, ownerID, categoryID string) ([]Pin, error) {
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(s.pinsTable),
		KeyConditionExpression:    aws.String("ownerId = :o"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":o": attrS(ownerID)},
	}
	if categoryID != "" {
		input.KeyConditionExpression = aws.String("ownerId = :o AND begins_with(sk, :p)")
		input.ExpressionAttributeValues[":p"] = attrS(categoryID + "#")
	}
	return queryAll[Pin](ctx, s.client, input)
}

func (s *DynamoStore) GetPin(ctx context.Context, ownerID, categoryID, pinID string) (Pin, error) {
	return getItem[Pin](ctx, s.client, s.pinsTable, pinKey(ownerID, categoryID, pinID))
}

func (s *DynamoStore) CreatePin(ctx context.Context, p Pin) error {
	return s.put(ctx, s.pinsTable, p, "attribute_not_exists(ownerId)", ErrConflict)
}

func (s *DynamoStore) UpdatePin(ctx context.Context, p Pin) error {
	return s.put(ctx, s.pinsTable, p, "attribute_exists(ownerId)", ErrNotFound)
}

func (s *DynamoStore) DeletePin(ctx context.Context, ownerID, categoryID, pinID string) error {
	return s.delete(ctx, s.pinsTable, pinKey(ownerID, categoryID, pinID))
}

// put は条件付きで項目を書き込む。条件を満たさなければ onFail を返す。
func (s *DynamoStore) put(ctx context.Context, table string, item any, condition string, onFail error) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(table),
		Item:                av,
		ConditionExpression: aws.String(condition),
	})
	if isConditionFailed(err) {
		return onFail
	}
	return err
}

// delete は項目を消す。項目がなければ ErrNotFound を返す。
func (s *DynamoStore) delete(ctx context.Context, table string, key map[string]types.AttributeValue) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:           aws.String(table),
		Key:                 key,
		ConditionExpression: aws.String("attribute_exists(ownerId)"),
	})
	if isConditionFailed(err) {
		return ErrNotFound
	}
	return err
}

func isConditionFailed(err error) bool {
	var ccf *types.ConditionalCheckFailedException
	return errors.As(err, &ccf)
}

func getItem[T any](ctx context.Context, client *dynamodb.Client, table string, key map[string]types.AttributeValue) (T, error) {
	var v T
	out, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(table),
		Key:            key,
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return v, err
	}
	if out.Item == nil {
		return v, ErrNotFound
	}
	err = attributevalue.UnmarshalMap(out.Item, &v)
	return v, err
}

// queryAll はページをたどって、クエリの結果を全部返す。
func queryAll[T any](ctx context.Context, client *dynamodb.Client, input *dynamodb.QueryInput) ([]T, error) {
	var out []T
	pages := dynamodb.NewQueryPaginator(client, input)
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		var items []T
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &items); err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}
