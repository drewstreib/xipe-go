package db

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

type DBInterface interface {
	PutRedirect(redirect *RedirectRecord) error
	GetRedirect(code string) (*RedirectRecord, error)
	GetCacheSize() int
}

type DynamoDBClient struct {
	client *dynamodb.Client
	table  string
	cache  *expirable.LRU[string, *CachedRedirect]
}

// CachedRedirect holds the redirect URL and original DynamoDB TTL
type CachedRedirect struct {
	URL       string
	DynamoTTL int64  // Original DynamoDB TTL timestamp
	Created   int64  // Creation timestamp
	IP        string // Creator IP address
}

type RedirectRecord struct {
	Code    string `dynamodbav:"code"`
	Typ     string `dynamodbav:"typ"`
	Val     string `dynamodbav:"val"`
	Ettl    int64  `dynamodbav:"ettl,omitempty"`
	Created int64  `dynamodbav:"created"`
	IP      string `dynamodbav:"ip"`
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

	// Initialize cache with configurable size
	cacheSize := 10000 // Default
	if envSize := os.Getenv("CACHE_SIZE"); envSize != "" {
		if size, err := strconv.Atoi(envSize); err == nil && size > 0 {
			cacheSize = size
		}
	}

	// Cache TTL is 1 hour
	cacheTTL := time.Hour
	cache := expirable.NewLRU[string, *CachedRedirect](cacheSize, nil, cacheTTL)

	log.Printf("Initialized LRU cache with size: %d, TTL: %v", cacheSize, cacheTTL)

	client := &DynamoDBClient{
		client: dynamodb.NewFromConfig(cfg),
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
				Typ:     "R",
				Val:     cached.URL,
				Ettl:    cached.DynamoTTL,
				Created: cached.Created,
				IP:      cached.IP,
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
	cached := &CachedRedirect{
		URL:       record.Val,
		DynamoTTL: record.Ettl,
		Created:   record.Created,
		IP:        record.IP,
	}
	d.cache.Add(code, cached)
	log.Printf("Cached redirect for code %s", code)

	return &record, nil
}

func (d *DynamoDBClient) GetCacheSize() int {
	return d.cache.Len()
}
