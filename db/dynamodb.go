package db

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type DBInterface interface {
	PutURL(key, url string) error
	GetURL(key string) (string, error)
}

type DynamoDBClient struct {
	client *dynamodb.DynamoDB
	table  string
}

type URLRecord struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

var DB DBInterface

func Init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))

	DB = &DynamoDBClient{
		client: dynamodb.New(sess),
		table:  "xipe-urls",
	}
}

func (d *DynamoDBClient) PutURL(key, url string) error {
	record := URLRecord{
		Key: key,
		URL: url,
	}

	av, err := dynamodbattribute.MarshalMap(record)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(d.table),
	}

	_, err = d.client.PutItem(input)
	return err
}

func (d *DynamoDBClient) GetURL(key string) (string, error) {
	result, err := d.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(d.table),
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String(key),
			},
		},
	})

	if err != nil {
		return "", err
	}

	if result.Item == nil {
		return "", nil
	}

	var record URLRecord
	err = dynamodbattribute.UnmarshalMap(result.Item, &record)
	if err != nil {
		return "", err
	}

	return record.URL, nil
}