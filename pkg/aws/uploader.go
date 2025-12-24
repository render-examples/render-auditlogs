package aws

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/renderinc/render-auditlogs/pkg/auditlogs"
	"github.com/renderinc/render-auditlogs/pkg/render"
)

type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type UploaderOptions struct {
	UseKMS           bool
	KMSKeyID         string
	BucketKeyEnabled bool
}

type Uploader struct {
	client S3Client
	bucket string
	opts   UploaderOptions
}

func NewUploader(ctx context.Context, client S3Client, bucket, region string) (*Uploader, error) {
	return NewUploaderWithOptions(ctx, client, bucket, region, UploaderOptions{})
}

func NewUploaderWithOptions(ctx context.Context, client S3Client, bucket, region string, opts UploaderOptions) (*Uploader, error) {
	return &Uploader{
		client: client,
		bucket: bucket,
		opts:   opts,
	}, nil
}

// UploadAuditLogs uploads audit logs to S3 with partitioned path structure
// Path format: workspace={workspaceID}/year={year}/month={month}/day={day}/hour={hour}/audit-logs-{timestamp}.json.gz
func (u *Uploader) UploadAuditLogs(ctx context.Context, auditLogType auditlogs.LogType, id string, data []render.AuditLogEntry) (string, error) {
	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Compress data with gzip
	var compressedData bytes.Buffer
	gzWriter := gzip.NewWriter(&compressedData)
	if _, err := gzWriter.Write(jsonData); err != nil {
		return "", fmt.Errorf("error compressing data: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return "", fmt.Errorf("error closing gzip writer: %w", err)
	}

	// Generate S3 key with partitioned structure
	key := generateS3Key(auditLogType, id, data[0].AuditLog.Timestamp)

	// Upload to S3
	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(compressedData.Bytes()),
		ContentType: aws.String("application/gzip"),
	}

	// Configure server-side encryption
	if u.opts.UseKMS {
		putInput.ServerSideEncryption = types.ServerSideEncryptionAwsKms
		if u.opts.KMSKeyID != "" {
			putInput.SSEKMSKeyId = aws.String(u.opts.KMSKeyID)
		}
		if u.opts.BucketKeyEnabled {
			putInput.BucketKeyEnabled = aws.Bool(true)
		}
	} else {
		// Default to SSE-S3 (AES256)
		putInput.ServerSideEncryption = types.ServerSideEncryptionAes256
	}

	_, err = u.client.PutObject(ctx, putInput)
	if err != nil {
		return "", fmt.Errorf("error uploading to S3: %w", err)
	}

	s3URI := fmt.Sprintf("s3://%s/%s", u.bucket, key)
	return s3URI, nil
}

// generateS3Key creates the partitioned S3 key
// Format: workspace={workspaceID}/year={year}/month={month}/day={day}/audit-logs-{timestamp}.json.gz
func generateS3Key(auditLogType auditlogs.LogType, id string, timestamp time.Time) string {
	filename := fmt.Sprintf("audit-logs-%s.json.gz", timestamp.Format("2006-01-02_15-04-05"))

	return fmt.Sprintf(
		"%s=%s/year=%d/month=%d/day=%d/%s",
		auditLogType,
		id,
		timestamp.Year(),
		int(timestamp.Month()),
		timestamp.Day(),
		filename,
	)
}
