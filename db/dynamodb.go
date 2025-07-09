package db

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type DBInterface interface {
	PutRedirect(redirect *RedirectRecord) error
	GetRedirect(code string) (*RedirectRecord, error)
}

type DynamoDBClient struct {
	client *dynamodb.DynamoDB
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
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))

	DB = &DynamoDBClient{
		client: dynamodb.New(sess),
		table:  "xipe_redirects",
	}
}

func (d *DynamoDBClient) PutRedirect(redirect *RedirectRecord) error {
	av, err := dynamodbattribute.MarshalMap(redirect)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(d.table),
		ConditionExpression: aws.String("attribute_not_exists(code)"),
	}

	_, err = d.client.PutItem(input)
	return err
}

func (d *DynamoDBClient) GetRedirect(code string) (*RedirectRecord, error) {
	result, err := d.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(d.table),
		Key: map[string]*dynamodb.AttributeValue{
			"code": {
				S: aws.String(code),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var record RedirectRecord
	err = dynamodbattribute.UnmarshalMap(result.Item, &record)
	if err != nil {
		return nil, err
	}

	return &record, nil
}