// Package storage provides an S3-compatible blob store client. Backed by
// MinIO in dev; AWS S3 / GCS / Azure Blob in hosted environments.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	s3     *s3.Client
	bucket string
}

type Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UsePathStyle    bool
}

func ConfigFromEnv() Config {
	return Config{
		Endpoint:        envOr("PACTLINE_S3_ENDPOINT", "http://localhost:9000"),
		Region:          envOr("PACTLINE_S3_REGION", "us-east-1"),
		AccessKeyID:     envOr("PACTLINE_S3_ACCESS_KEY", "pactline"),
		SecretAccessKey: envOr("PACTLINE_S3_SECRET_KEY", "pactlinepactline"),
		Bucket:          envOr("PACTLINE_S3_BUCKET", "pactline-documents"),
		UsePathStyle:    true,
	}
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func New(ctx context.Context, c Config) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	endpoint := c.Endpoint
	cli := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = c.UsePathStyle
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	out := &Client{s3: cli, bucket: c.Bucket}
	if err := out.ensureBucket(ctx); err != nil {
		return nil, fmt.Errorf("ensure bucket: %w", err)
	}
	return out, nil
}

func (c *Client) ensureBucket(ctx context.Context) error {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &c.bucket})
	if err == nil {
		return nil
	}
	_, err = c.s3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &c.bucket})
	if err != nil {
		if _, e2 := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &c.bucket}); e2 == nil {
			return nil
		}
		return err
	}
	return nil
}

// Put uploads body to key.
func (c *Client) Put(ctx context.Context, key, mimeType string, body []byte) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucket,
		Key:         &key,
		Body:        bytes.NewReader(body),
		ContentType: &mimeType,
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

// Get returns the full object body.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := c.s3.GetObject(ctx, &s3.GetObjectInput{Bucket: &c.bucket, Key: &key})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	defer obj.Body.Close()
	return io.ReadAll(obj.Body)
}

// SignedURL returns a presigned GET URL valid for ttl.
func (c *Client) SignedURL(ctx context.Context, key string, ttl time.Duration) (*url.URL, error) {
	pre := s3.NewPresignClient(c.s3)
	req, err := pre.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &c.bucket, Key: &key},
		s3.WithPresignExpires(ttl))
	if err != nil {
		return nil, fmt.Errorf("presign: %w", err)
	}
	return url.Parse(req.URL)
}
