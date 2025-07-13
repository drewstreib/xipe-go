package db

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/compress/zstd"
)

// S3Interface defines the interface for S3 operations
type S3Interface interface {
	PutObject(key string, data []byte) error
	GetObject(key string) ([]byte, error)
}

// S3Client implements S3Interface for real S3 operations
type S3Client struct {
	client  *s3.Client
	bucket  string
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

// NewS3Client creates a new S3 client
func NewS3Client() (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		return nil, err
	}

	// Create zstd encoder with level 3 compression
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevel(3)))
	if err != nil {
		return nil, err
	}

	// Create zstd decoder
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	return &S3Client{
		client:  s3.NewFromConfig(cfg),
		bucket:  "xipe-data",
		encoder: encoder,
		decoder: decoder,
	}, nil
}

// PutObject stores compressed data in S3
func (s *S3Client) PutObject(key string, data []byte) error {
	// Compress data using zstd level 3
	compressedData := s.encoder.EncodeAll(data, make([]byte, 0, len(data)))

	// Calculate compression ratio for logging
	originalSize := len(data)
	compressedSize := len(compressedData)
	compressionRatio := float64(originalSize-compressedSize) / float64(originalSize) * 100

	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(compressedData),
	})
	if err != nil {
		log.Printf("Failed to put object %s to S3: %v", key, err)
		return err
	}
	log.Printf("Successfully stored object %s in S3 (compressed %d bytes to %d bytes, %.1f%% reduction)",
		key, originalSize, compressedSize, compressionRatio)
	return nil
}

// GetObject retrieves and decompresses data from S3
func (s *S3Client) GetObject(key string) ([]byte, error) {
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Failed to get object %s from S3: %v", key, err)
		return nil, err
	}
	defer func() {
		if closeErr := result.Body.Close(); closeErr != nil {
			log.Printf("Failed to close S3 response body: %v", closeErr)
		}
	}()

	// Read compressed data from S3
	compressedData, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Failed to read object %s body from S3: %v", key, err)
		return nil, err
	}

	// Decompress data using zstd
	decompressedData, err := s.decoder.DecodeAll(compressedData, nil)
	if err != nil {
		log.Printf("Failed to decompress object %s from S3: %v", key, err)
		return nil, err
	}

	// Calculate compression info for logging
	compressedSize := len(compressedData)
	decompressedSize := len(decompressedData)
	compressionRatio := float64(decompressedSize-compressedSize) / float64(decompressedSize) * 100

	log.Printf("Successfully retrieved object %s from S3 (decompressed %d bytes to %d bytes, %.1f%% reduction)",
		key, compressedSize, decompressedSize, compressionRatio)
	return decompressedData, nil
}
