package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "github.com/grigory51/podvid/internal/config"
)

type S3Client struct {
	client *s3.Client
	bucket string
}

func NewS3Client(cfg *appconfig.Config) (*S3Client, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.S3.AccessKey, cfg.S3.SecretKey, ""),
		),
	}

	if cfg.S3.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.S3.Region))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		},
	}
	if cfg.S3.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.S3.Endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)
	return &S3Client{client: client, bucket: cfg.S3.Bucket}, nil
}

func (c *S3Client) Upload(ctx context.Context, key string, data io.Reader, contentType string) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        data,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("uploading %s: %w", key, err)
	}
	return nil
}

func (c *S3Client) UploadFile(ctx context.Context, key string, data []byte, contentType string) error {
	return c.Upload(ctx, key, bytes.NewReader(data), contentType)
}

func (c *S3Client) Download(ctx context.Context, key string) ([]byte, error) {
	resp, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", key, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", key, err)
	}
	return data, nil
}

func (c *S3Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting %s: %w", key, err)
	}
	return nil
}

func (c *S3Client) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing objects with prefix %s: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			keys = append(keys, aws.ToString(obj.Key))
		}
	}
	return keys, nil
}

func (c *S3Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, fmt.Errorf("checking %s: %w", key, err)
	}
	return true, nil
}

func (c *S3Client) DeletePrefix(ctx context.Context, prefix string) error {
	keys, err := c.ListObjects(ctx, prefix)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := c.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}
