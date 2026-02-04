package database

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds configuration for connecting to S3-compatible storage
type S3Config struct {
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	UseSSL          bool   `json:"use_ssl"`
}

// S3ConnectionResult holds the result of a connection test
type S3ConnectionResult struct {
	Success      bool     `json:"success"`
	Buckets      []string `json:"buckets,omitempty"`
	ResponseTime float64  `json:"response_time_ms,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// S3Bucket represents a bucket in S3
type S3Bucket struct {
	Name         string `json:"name"`
	CreationDate string `json:"creation_date"`
}

// S3Object represents an object in S3
type S3Object struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
	ETag         string `json:"etag"`
	IsDir        bool   `json:"is_dir"`
}

// S3ObjectInfo holds detailed information about an S3 object
type S3ObjectInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	LastModified string            `json:"last_modified"`
	ETag         string            `json:"etag"`
	Metadata     map[string]string `json:"metadata"`
}

// createS3Client creates a new MinIO client for S3 operations
func createS3Client(config S3Config) (*minio.Client, error) {
	return minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
		Region: config.Region,
	})
}

// TestS3Connection tests connectivity to S3-compatible storage
func TestS3Connection(ctx context.Context, config S3Config) S3ConnectionResult {
	start := time.Now()

	client, err := createS3Client(config)
	if err != nil {
		return S3ConnectionResult{
			Success: false,
			Error:   "Failed to create client: " + err.Error(),
		}
	}

	// List buckets to verify connection
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return S3ConnectionResult{
			Success: false,
			Error:   "Failed to connect: " + err.Error(),
		}
	}

	bucketNames := make([]string, len(buckets))
	for i, b := range buckets {
		bucketNames[i] = b.Name
	}

	return S3ConnectionResult{
		Success:      true,
		Buckets:      bucketNames,
		ResponseTime: float64(time.Since(start).Milliseconds()),
	}
}

// ListS3Buckets lists all buckets
func ListS3Buckets(ctx context.Context, config S3Config) ([]S3Bucket, error) {
	client, err := createS3Client(config)
	if err != nil {
		return nil, err
	}

	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]S3Bucket, len(buckets))
	for i, b := range buckets {
		result[i] = S3Bucket{
			Name:         b.Name,
			CreationDate: b.CreationDate.Format(time.RFC3339),
		}
	}

	return result, nil
}

// CreateS3Bucket creates a new bucket
func CreateS3Bucket(ctx context.Context, config S3Config, bucket string) error {
	client, err := createS3Client(config)
	if err != nil {
		return err
	}

	return client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{
		Region: config.Region,
	})
}

// DeleteS3Bucket deletes a bucket (must be empty)
func DeleteS3Bucket(ctx context.Context, config S3Config, bucket string) error {
	client, err := createS3Client(config)
	if err != nil {
		return err
	}

	return client.RemoveBucket(ctx, bucket)
}

// ListS3Objects lists objects in a bucket with optional prefix
func ListS3Objects(ctx context.Context, config S3Config, bucket, prefix string, maxKeys int) ([]S3Object, error) {
	client, err := createS3Client(config)
	if err != nil {
		return nil, err
	}

	// Normalize prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	if prefix == "/" {
		prefix = ""
	}

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
		MaxKeys:   maxKeys,
	}

	var objects []S3Object
	dirs := make(map[string]bool)

	for obj := range client.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return nil, obj.Err
		}

		key := obj.Key

		// Check if this is a "directory" (common prefix)
		if strings.HasSuffix(key, "/") {
			// It's a folder
			dirName := strings.TrimPrefix(key, prefix)
			dirName = strings.TrimSuffix(dirName, "/")
			if dirName != "" && !dirs[dirName] {
				dirs[dirName] = true
				objects = append(objects, S3Object{
					Key:   key,
					IsDir: true,
				})
			}
		} else {
			// It's a file - check if it has nested path
			relKey := strings.TrimPrefix(key, prefix)
			if idx := strings.Index(relKey, "/"); idx > 0 {
				// This is inside a subfolder
				dirName := relKey[:idx]
				if !dirs[dirName] {
					dirs[dirName] = true
					objects = append(objects, S3Object{
						Key:   prefix + dirName + "/",
						IsDir: true,
					})
				}
			} else {
				// Direct file
				objects = append(objects, S3Object{
					Key:          key,
					Size:         obj.Size,
					LastModified: obj.LastModified.Format(time.RFC3339),
					ETag:         strings.Trim(obj.ETag, "\""),
					IsDir:        false,
				})
			}
		}

		if maxKeys > 0 && len(objects) >= maxKeys {
			break
		}
	}

	return objects, nil
}

// GetS3ObjectInfo gets detailed information about an object
func GetS3ObjectInfo(ctx context.Context, config S3Config, bucket, key string) (*S3ObjectInfo, error) {
	client, err := createS3Client(config)
	if err != nil {
		return nil, err
	}

	stat, err := client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	for k, v := range stat.UserMetadata {
		metadata[k] = v
	}

	return &S3ObjectInfo{
		Key:          key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified.Format(time.RFC3339),
		ETag:         strings.Trim(stat.ETag, "\""),
		Metadata:     metadata,
	}, nil
}

// UploadS3Object uploads data to S3
func UploadS3Object(ctx context.Context, config S3Config, bucket, key string, data []byte, contentType string) error {
	client, err := createS3Client(config)
	if err != nil {
		return err
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	reader := bytes.NewReader(data)
	_, err = client.PutObject(ctx, bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})

	return err
}

// DownloadS3Object downloads an object from S3
func DownloadS3Object(ctx context.Context, config S3Config, bucket, key string) ([]byte, string, error) {
	client, err := createS3Client(config)
	if err != nil {
		return nil, "", err
	}

	obj, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		return nil, "", err
	}

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, "", err
	}

	return data, stat.ContentType, nil
}

// DeleteS3Object deletes an object from S3
func DeleteS3Object(ctx context.Context, config S3Config, bucket, key string) error {
	client, err := createS3Client(config)
	if err != nil {
		return err
	}

	return client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// GetPresignedURL generates a presigned URL for downloading an object
func GetPresignedURL(ctx context.Context, config S3Config, bucket, key string, expiry time.Duration) (string, error) {
	client, err := createS3Client(config)
	if err != nil {
		return "", err
	}

	url, err := client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}

	return url.String(), nil
}
