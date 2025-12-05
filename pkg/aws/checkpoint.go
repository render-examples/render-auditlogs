package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/renderinc/render-auditlogs/pkg/auditlogs"
)

// Checkpoint represents the state to persist between runs
type Checkpoint struct {
	LastCursor    string    `json:"lastCursor"`
	LastTimestamp time.Time `json:"lastTimestamp"`
}

const checkpointKey = "checkpoint.json"

// LoadCheckpoint reads the checkpoint from S3. Returns nil if file doesn't exist.
func (u *Uploader) LoadCheckpoint(ctx context.Context, logType auditlogs.LogType, workspace string) (*Checkpoint, error) {
	result, err := u.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(fmt.Sprintf("%s=%s/%s", logType, workspace, checkpointKey)),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			// No checkpoint file exists yet, return nil
			return nil, nil
		}
		return nil, fmt.Errorf("error reading checkpoint from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading checkpoint body: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("error unmarshaling checkpoint: %w", err)
	}

	return &cp, nil
}

// SaveCheckpoint writes the checkpoint to S3
func (u *Uploader) SaveCheckpoint(ctx context.Context, cp *Checkpoint, logType auditlogs.LogType, workspace string) error {
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling checkpoint: %w", err)
	}

	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(u.bucket),
		Key:                  aws.String(fmt.Sprintf("%s=%s/%s", logType, workspace, checkpointKey)),
		Body:                 bytes.NewReader(data),
		ContentType:          aws.String("application/json"),
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	})
	if err != nil {
		return fmt.Errorf("error writing checkpoint to S3: %w", err)
	}

	return nil
}
