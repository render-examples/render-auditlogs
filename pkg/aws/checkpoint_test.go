package aws_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"

	"github.com/renderinc/render-auditlogs/pkg/auditlogs"
	awspkg "github.com/renderinc/render-auditlogs/pkg/aws"
)

type mockS3Client struct {
	getObjectFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	putObjectFunc func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.getObjectFunc(ctx, params, optFns...)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m.putObjectFunc(ctx, params, optFns...)
}

func TestLoadCheckpoint(t *testing.T) {
	ctx := context.Background()
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("successfully loads checkpoint", func(t *testing.T) {
		checkpoint := awspkg.Checkpoint{
			LastCursor:    "test-cursor-123",
			LastTimestamp: testTime,
		}
		checkpointJSON, err := json.Marshal(checkpoint)
		require.NoError(t, err)

		s3Client := &mockS3Client{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				require.Equal(t, "test-bucket", *params.Bucket)
				require.Equal(t, "workspace=test-workspace/checkpoint.json", *params.Key)

				return &s3.GetObjectOutput{
					Body: io.NopCloser(bytes.NewReader(checkpointJSON)),
				}, nil
			},
		}

		uploader, err := awspkg.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		cp, err := uploader.LoadCheckpoint(ctx, auditlogs.WorkspaceAuditLog, "test-workspace")

		require.NoError(t, err)
		require.NotNil(t, cp)
		require.Equal(t, "test-cursor-123", cp.LastCursor)
		require.Equal(t, testTime, cp.LastTimestamp)
	})

	t.Run("returns nil when checkpoint does not exist", func(t *testing.T) {
		s3Client := &mockS3Client{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, &types.NoSuchKey{
					Message: aws.String("The specified key does not exist"),
				}
			},
		}

		uploader, err := awspkg.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		cp, err := uploader.LoadCheckpoint(ctx, auditlogs.WorkspaceAuditLog, "test-workspace")

		require.NoError(t, err)
		require.Nil(t, cp)
	})

	t.Run("returns error on S3 error", func(t *testing.T) {
		s3Client := &mockS3Client{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("S3 connection error")
			},
		}

		uploader, err := awspkg.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		cp, err := uploader.LoadCheckpoint(ctx, auditlogs.WorkspaceAuditLog, "test-workspace")

		require.Error(t, err)
		require.Nil(t, cp)
		require.Contains(t, err.Error(), "error reading checkpoint from S3")
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		s3Client := &mockS3Client{
			getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body: io.NopCloser(bytes.NewReader([]byte("invalid json"))),
				}, nil
			},
		}

		uploader, err := awspkg.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		cp, err := uploader.LoadCheckpoint(ctx, auditlogs.WorkspaceAuditLog, "test-workspace")

		require.Error(t, err)
		require.Nil(t, cp)
		require.Contains(t, err.Error(), "error unmarshaling checkpoint")
	})
}

func TestSaveCheckpoint(t *testing.T) {
	ctx := context.Background()
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("successfully saves checkpoint", func(t *testing.T) {
		checkpoint := &awspkg.Checkpoint{
			LastCursor:    "test-cursor-456",
			LastTimestamp: testTime,
		}

		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				require.Equal(t, "test-bucket", *params.Bucket)
				require.Equal(t, "workspace=test-workspace/checkpoint.json", *params.Key)
				require.Equal(t, "application/json", *params.ContentType)

				// Read and verify the body content
				bodyBytes, err := io.ReadAll(params.Body)
				require.NoError(t, err)

				var savedCP awspkg.Checkpoint
				err = json.Unmarshal(bodyBytes, &savedCP)
				require.NoError(t, err)
				require.Equal(t, checkpoint.LastCursor, savedCP.LastCursor)
				require.Equal(t, checkpoint.LastTimestamp, savedCP.LastTimestamp)

				return &s3.PutObjectOutput{}, nil
			},
		}

		uploader, err := awspkg.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		err = uploader.SaveCheckpoint(ctx, checkpoint, auditlogs.WorkspaceAuditLog, "test-workspace")

		require.NoError(t, err)
	})

	t.Run("uses KMS with key ID and bucket key enabled", func(t *testing.T) {
		checkpoint := &awspkg.Checkpoint{
			LastCursor:    "kms-cursor",
			LastTimestamp: testTime,
		}

		const kmsKey = "arn:aws:kms:us-west-2:123456789012:key/abcdefab-1234-5678-9abc-def012345678"

		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				require.Equal(t, "test-bucket", *params.Bucket)
				require.Equal(t, "workspace=test-workspace/checkpoint.json", *params.Key)
				require.Equal(t, types.ServerSideEncryptionAwsKms, params.ServerSideEncryption)
				require.NotNil(t, params.SSEKMSKeyId)
				require.Equal(t, kmsKey, *params.SSEKMSKeyId)
				require.NotNil(t, params.BucketKeyEnabled)
				require.True(t, *params.BucketKeyEnabled)
				return &s3.PutObjectOutput{}, nil
			},
		}

		uploader, err := awspkg.NewUploaderWithOptions(ctx, s3Client, "test-bucket", "test-region", awspkg.UploaderOptions{
			UseKMS:           true,
			KMSKeyID:         kmsKey,
			BucketKeyEnabled: true,
		})
		require.NoError(t, err)

		err = uploader.SaveCheckpoint(ctx, checkpoint, auditlogs.WorkspaceAuditLog, "test-workspace")
		require.NoError(t, err)
	})

	t.Run("returns error on S3 error", func(t *testing.T) {
		checkpoint := &awspkg.Checkpoint{
			LastCursor:    "test-cursor",
			LastTimestamp: testTime,
		}

		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("S3 write error")
			},
		}

		uploader, err := awspkg.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		err = uploader.SaveCheckpoint(ctx, checkpoint, auditlogs.WorkspaceAuditLog, "test-workspace")

		require.Error(t, err)
		require.Contains(t, err.Error(), "error writing checkpoint to S3")
	})
}
