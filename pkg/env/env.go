package env

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	"github.com/renderinc/render-auditlogs/pkg/logger"
)

type Config struct {
	WorkspaceIDS       []string `required:"true" split_words:"true"`
	OrganizationID     string   `required:"false" split_words:"true"`
	S3Bucket           string   `required:"true" split_words:"true"`
	S3BucketKeyEnabled bool     `required:"false" split_words:"true"`
	S3KMSKeyID         string   `required:"false" split_words:"true"`
	S3UseKMS           bool     `required:"false" split_words:"true"`
	RenderAPIKey       string   `required:"true" split_words:"true"`
	AWSAccessKeyID     string   `required:"true" split_words:"true"`
	AWSSecretAccessKey string   `required:"true" split_words:"true"`
	AWSRegion          string   `required:"true" split_words:"true"`

	AWSConfig aws.Config
}

func LoadConfig(ctx context.Context, config *Config) error {
	logger.FromContext(ctx).Info("Loading config")

	if os.Getenv("LOCAL") != "false" {
		if err := loadEnvironmentFiles(); err != nil {
			return err
		}
	}

	if err := envconfig.Process("", config); err != nil {
		return err
	}

	awscfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.AWSRegion))
	if err != nil {
		return err
	}

	config.AWSConfig = awscfg

	return nil
}

func loadEnvironmentFiles() error {
	if err := godotenv.Load(".env"); err != nil {
		return err
	}

	return nil
}
