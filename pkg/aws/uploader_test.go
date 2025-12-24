package aws_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"

	"github.com/renderinc/render-auditlogs/pkg/auditlogs"
	"github.com/renderinc/render-auditlogs/pkg/aws"
	"github.com/renderinc/render-auditlogs/pkg/render"
	"github.com/renderinc/render-auditlogs/pkg/testhelpers"
)

func TestUploadAuditLogs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	testData := testhelpers.CreateTestAuditLogs(3, time.Date(testTime.Year(), testTime.Month(), testTime.Day(), 0, 0, 0, 0, time.UTC))

	t.Run("successfully uploads audit logs for workspace", func(t *testing.T) {
		t.Parallel()
		var capturedBody []byte

		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				require.Equal(t, "test-bucket", *params.Bucket)
				require.Equal(t, "application/gzip", *params.ContentType)

				var err error
				capturedBody, err = io.ReadAll(params.Body)
				require.NoError(t, err)

				return &s3.PutObjectOutput{}, nil
			},
		}

		uploader, err := aws.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		s3URI, err := uploader.UploadAuditLogs(ctx, auditlogs.WorkspaceAuditLog, "workspace-123", testData)

		require.NoError(t, err)
		require.NotEmpty(t, s3URI)

		// Verify S3 URI format
		require.True(t, strings.HasPrefix(s3URI, "s3://test-bucket/"))
		require.Contains(t, s3URI, "workspace=workspace-123/year=2024/month=1/day=15/audit-logs-2024-01-15")

		// Verify the data was compressed
		gzReader, err := gzip.NewReader(bytes.NewReader(capturedBody))
		require.NoError(t, err)
		defer gzReader.Close()

		decompressed, err := io.ReadAll(gzReader)
		require.NoError(t, err)

		var uploadedData []render.AuditLogEntry
		err = json.Unmarshal(decompressed, &uploadedData)
		require.NoError(t, err)
		require.Equal(t, testData, uploadedData)
	})

	t.Run("successfully uploads audit logs for organization", func(t *testing.T) {
		t.Parallel()
		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return &s3.PutObjectOutput{}, nil
			},
		}

		uploader, err := aws.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		s3URI, err := uploader.UploadAuditLogs(ctx, auditlogs.OrganizationAuditLog, "org-456", testData)

		require.NoError(t, err)
		require.NotEmpty(t, s3URI)

		// Verify organization log type is used in the path
		require.Contains(t, s3URI, "organization=org-456")
	})

	t.Run("returns error on S3 upload failure", func(t *testing.T) {
		t.Parallel()
		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("S3 upload failed")
			},
		}

		uploader, err := aws.NewUploader(ctx, s3Client, "test-bucket", "test-region")
		require.NoError(t, err)

		s3URI, err := uploader.UploadAuditLogs(ctx, auditlogs.WorkspaceAuditLog, "workspace-123", testData)

		require.Error(t, err)
		require.Contains(t, err.Error(), "error uploading to S3")
		require.Empty(t, s3URI)
	})

	t.Run("uses default SSE-S3 encryption when KMS not enabled", func(t *testing.T) {
		t.Parallel()
		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				require.Equal(t, "test-bucket", *params.Bucket)
				require.Equal(t, types.ServerSideEncryptionAes256, params.ServerSideEncryption)
				require.Nil(t, params.SSEKMSKeyId)
				require.Nil(t, params.BucketKeyEnabled)
				return &s3.PutObjectOutput{}, nil
			},
		}

		uploader, err := aws.NewUploaderWithOptions(ctx, s3Client, "test-bucket", "test-region", aws.UploaderOptions{
			UseKMS: false,
		})
		require.NoError(t, err)

		s3URI, err := uploader.UploadAuditLogs(ctx, auditlogs.WorkspaceAuditLog, "workspace-123", testData)

		require.NoError(t, err)
		require.NotEmpty(t, s3URI)
	})

	t.Run("uses KMS encryption without specific key ID", func(t *testing.T) {
		t.Parallel()
		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				require.Equal(t, "test-bucket", *params.Bucket)
				require.Equal(t, types.ServerSideEncryptionAwsKms, params.ServerSideEncryption)
				require.Nil(t, params.SSEKMSKeyId)
				require.Nil(t, params.BucketKeyEnabled)
				return &s3.PutObjectOutput{}, nil
			},
		}

		uploader, err := aws.NewUploaderWithOptions(ctx, s3Client, "test-bucket", "test-region", aws.UploaderOptions{
			UseKMS: true,
		})
		require.NoError(t, err)

		s3URI, err := uploader.UploadAuditLogs(ctx, auditlogs.WorkspaceAuditLog, "workspace-123", testData)

		require.NoError(t, err)
		require.NotEmpty(t, s3URI)
	})

	t.Run("uses KMS encryption with specific key ID", func(t *testing.T) {
		t.Parallel()
		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				require.Equal(t, "test-bucket", *params.Bucket)
				require.Equal(t, types.ServerSideEncryptionAwsKms, params.ServerSideEncryption)
				require.NotNil(t, params.SSEKMSKeyId)
				require.Equal(t, "arn:aws:kms:us-west-2:123456789012:key/12345678-1234-1234-1234-123456789012", *params.SSEKMSKeyId)
				require.Nil(t, params.BucketKeyEnabled)
				return &s3.PutObjectOutput{}, nil
			},
		}

		uploader, err := aws.NewUploaderWithOptions(ctx, s3Client, "test-bucket", "test-region", aws.UploaderOptions{
			UseKMS:   true,
			KMSKeyID: "arn:aws:kms:us-west-2:123456789012:key/12345678-1234-1234-1234-123456789012",
		})
		require.NoError(t, err)

		s3URI, err := uploader.UploadAuditLogs(ctx, auditlogs.WorkspaceAuditLog, "workspace-123", testData)

		require.NoError(t, err)
		require.NotEmpty(t, s3URI)
	})

	t.Run("uses KMS encryption with bucket key enabled", func(t *testing.T) {
		t.Parallel()
		s3Client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				require.Equal(t, "test-bucket", *params.Bucket)
				require.Equal(t, types.ServerSideEncryptionAwsKms, params.ServerSideEncryption)
				require.Nil(t, params.SSEKMSKeyId)
				require.NotNil(t, params.BucketKeyEnabled)
				require.True(t, *params.BucketKeyEnabled)
				return &s3.PutObjectOutput{}, nil
			},
		}

		uploader, err := aws.NewUploaderWithOptions(ctx, s3Client, "test-bucket", "test-region", aws.UploaderOptions{
			UseKMS:           true,
			BucketKeyEnabled: true,
		})
		require.NoError(t, err)

		s3URI, err := uploader.UploadAuditLogs(ctx, auditlogs.WorkspaceAuditLog, "workspace-123", testData)

		require.NoError(t, err)
		require.NotEmpty(t, s3URI)
	})
}
