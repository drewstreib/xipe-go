package db

import (
	"context"
	"log"

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
	Code string `dynamodbav:"code"`
	Typ  string `dynamodbav:"typ"`
	Val  string `dynamodbav:"val"`
	Ettl int64  `dynamodbav:"ettl,omitempty"`
}

func NewDynamoDBClient() (DBInterface, error) {
	log.Println("Initializing DynamoDB client...")

	// Log some environment info for debugging
	log.Printf("AWS Region: us-east-1")
	log.Printf("DynamoDB Table: xipe_redirects")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		return nil, err
	}

	// Try to get credentials to verify they're working
	creds, err := cfg.Credentials.Retrieve(context.TODO())
	if err != nil {
		log.Printf("Failed to retrieve AWS credentials: %v", err)
	} else {
		log.Printf("AWS credentials retrieved successfully - Source: %s", creds.Source)
		// Don't log the actual keys for security
	}

	client := &DynamoDBClient{
		client: dynamodb.NewFromConfig(cfg),
		table:  "xipe_redirects",
	}
	log.Printf("DynamoDB client initialized successfully for table: %s", "xipe_redirects")
	return client, nil
}

func (d *DynamoDBClient) PutRedirect(redirect *RedirectRecord) error {
	log.Printf("PutRedirect called with code: %s, table: %s", redirect.Code, d.table)
	av, err := attributevalue.MarshalMap(redirect)
	if err != nil {
		log.Printf("Failed to marshal redirect record: %v", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:                av,
		TableName:           aws.String(d.table),
		ConditionExpression: aws.String("attribute_not_exists(code)"),
	}

	_, err = d.client.PutItem(context.TODO(), input)
	if err != nil {
		log.Printf("DynamoDB PutItem failed: %v", err)
	}
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
