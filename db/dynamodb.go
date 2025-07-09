package db

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DBInterface interface {
	PutRedirect(redirect *RedirectRecord) error
	GetRedirect(code string) (*RedirectRecord, error)
}

type DynamoDBClient struct {
	client *dynamodb.Client
	table  string
}

type RedirectRecord struct {
	Code string `json:"code"`
	Typ  string `json:"typ"`
	Val  string `json:"val"`
	Ettl int64  `json:"ettl,omitempty"`
}

var DB DBInterface

func Init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		panic(err)
	}

	DB = &DynamoDBClient{
		client: dynamodb.NewFromConfig(cfg),
		table:  "xipe_redirects",
	}
}

func (d *DynamoDBClient) PutRedirect(redirect *RedirectRecord) error {
	av, err := attributevalue.MarshalMap(redirect)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:                av,
		TableName:           aws.String(d.table),
		ConditionExpression: aws.String("attribute_not_exists(code)"),
	}

	_, err = d.client.PutItem(context.TODO(), input)
	return err
}

func (d *DynamoDBClient) GetRedirect(code string) (*RedirectRecord, error) {
	result, err := d.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(d.table),
		Key: map[string]types.AttributeValue{
			"code": &types.AttributeValueMemberS{Value: code},
		},
	})

	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var record RedirectRecord
	err = attributevalue.UnmarshalMap(result.Item, &record)
	if err != nil {
		return nil, err
	}

	return &record, nil
}
