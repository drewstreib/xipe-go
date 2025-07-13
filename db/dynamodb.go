package db

import (
	"context"
	"log"
	"time"

	"github.com/drewstreib/xipe-go/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

type DBInterface interface {
	PutRedirect(redirect *RedirectRecord) error
	GetRedirect(code string) (*RedirectRecord, error)
	DeleteRedirect(code string, ownerID string) error
	GetCacheSize() int
}

type DynamoDBClient struct {
	client *dynamodb.Client
	table  string
	cache  *expirable.LRU[string, *CachedRecord]
}

// CachedRecord holds the data/URL and original DynamoDB TTL
type CachedRecord struct {
	Val       string // URL or data content
	Typ       string // "R" for redirect, "D" for data
	DynamoTTL int64  // Original DynamoDB TTL timestamp
	Created   int64  // Creation timestamp
	IP        string // Creator IP address
	Owner     string // Owner ID for deletion authentication
}

type RedirectRecord struct {
	Code    string `dynamodbav:"code"`
	Typ     string `dynamodbav:"typ"`
	Val     string `dynamodbav:"val"`
	Ettl    int64  `dynamodbav:"ettl,omitempty"`
	Created int64  `dynamodbav:"created"`
	IP      string `dynamodbav:"ip"`
	Owner   string `dynamodbav:"owner"`
}

func NewDynamoDBClient(cfg *config.Config) (DBInterface, error) {
	log.Println("Initializing DynamoDB client...")

	// Log some environment info for debugging
	log.Printf("AWS Region: us-east-1")
	log.Printf("DynamoDB Table: xipe_redirects")

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion("us-east-1"))
	if err != nil {
		log.Printf("Failed to load AWS config: %v", err)
		return nil, err
	}

	// Try to get credentials to verify they're working
	creds, err := awsCfg.Credentials.Retrieve(context.TODO())
	if err != nil {
		log.Printf("Failed to retrieve AWS credentials: %v", err)
	} else {
		log.Printf("AWS credentials retrieved successfully - Source: %s", creds.Source)
		// Don't log the actual keys for security
	}

	// Use cache max items from config
	cacheMaxItems := cfg.CacheMaxItems

	// Cache TTL is 1 hour
	cacheTTL := time.Hour
	cache := expirable.NewLRU[string, *CachedRecord](cacheMaxItems, nil, cacheTTL)

	log.Printf("Initialized LRU cache with max items: %d, TTL: %v", cacheMaxItems, cacheTTL)

	client := &DynamoDBClient{
		client: dynamodb.NewFromConfig(awsCfg),
		table:  "xipe_redirects",
		cache:  cache,
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
	// Check cache first
	if cached, found := d.cache.Get(code); found {
		// Check if the DynamoDB TTL is still valid
		if cached.DynamoTTL > 0 && time.Now().Unix() > cached.DynamoTTL {
			// DynamoDB item has expired, remove from cache and fall through to DB
			d.cache.Remove(code)
			log.Printf("Cache hit for code %s but DynamoDB TTL expired, evicting from cache", code)
		} else {
			// Cache hit with valid TTL, return cached value
			log.Printf("Cache hit for code %s", code)
			return &RedirectRecord{
				Code:    code,
				Typ:     cached.Typ,
				Val:     cached.Val,
				Ettl:    cached.DynamoTTL,
				Created: cached.Created,
				IP:      cached.IP,
				Owner:   cached.Owner,
			}, nil
		}
	}

	// Cache miss or expired, query DynamoDB
	log.Printf("Cache miss for code %s, querying DynamoDB", code)
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

	// Cache the result for 1 hour
	cached := &CachedRecord{
		Val:       record.Val,
		Typ:       record.Typ,
		DynamoTTL: record.Ettl,
		Created:   record.Created,
		IP:        record.IP,
		Owner:     record.Owner,
	}
	d.cache.Add(code, cached)
	log.Printf("Cached redirect for code %s", code)

	return &record, nil
}

func (d *DynamoDBClient) DeleteRedirect(code string, ownerID string) error {
	log.Printf("DeleteRedirect called with code: %s", code)

	// First, get the item to verify ownership
	record, err := d.GetRedirect(code)
	if err != nil {
		log.Printf("Failed to get redirect for ownership check: %v", err)
		return err
	}

	// Return same error for both "not found" and "wrong owner" for security
	if record == nil || record.Owner != ownerID {
		log.Printf("Delete failed: record not found or owner mismatch")
		return &types.ConditionalCheckFailedException{}
	}

	// Delete from DynamoDB with condition to double-check ownership
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(d.table),
		Key: map[string]types.AttributeValue{
			"code": &types.AttributeValueMemberS{Value: code},
		},
		ConditionExpression: aws.String("#owner = :owner"),
		ExpressionAttributeNames: map[string]string{
			"#owner": "owner",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":owner": &types.AttributeValueMemberS{Value: ownerID},
		},
	}

	_, err = d.client.DeleteItem(context.TODO(), input)
	if err != nil {
		log.Printf("DynamoDB DeleteItem failed: %v", err)
		return err
	}

	// Remove from cache
	d.cache.Remove(code)
	log.Printf("Successfully deleted redirect for code: %s", code)

	return nil
}

func (d *DynamoDBClient) GetCacheSize() int {
	return d.cache.Len()
}
