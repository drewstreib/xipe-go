package db

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Interface defines the interface for S3 operations
type S3Interface interface {
	PutObject(key string, data []byte) error
	GetObject(key string) ([]byte, error)
}

// S3Client implements S3Interface for real S3 operations
type S3Client struct {
	client *s3.Client
	bucket string
}

// NewS3Client creates a new S3 client
func NewS3Client() (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		return nil, err
	}

	return &S3Client{
		client: s3.NewFromConfig(cfg),
		bucket: "xipe-data",
	}, nil
}

// PutObject stores data in S3
func (s *S3Client) PutObject(key string, data []byte) error {
	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		log.Printf("Failed to put object %s to S3: %v", key, err)
		return err
	}
	log.Printf("Successfully stored object %s in S3", key)
	return nil
}

// GetObject retrieves data from S3
func (s *S3Client) GetObject(key string) ([]byte, error) {
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Failed to get object %s from S3: %v", key, err)
		return nil, err
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Failed to read object %s body from S3: %v", key, err)
		return nil, err
	}

	log.Printf("Successfully retrieved object %s from S3 (%d bytes)", key, len(data))
	return data, nil
}
