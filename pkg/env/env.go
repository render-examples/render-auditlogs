package env

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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
	AWSRegion          string   `required:"true" split_words:"true"`

	// If AWSWebIdentityTokenFile is set, OIDC is used. Otherwise the AWS SDK's
	// default credential chain falls back to AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY.
	AWSRoleARN string `required:"false" split_words:"true"`
	// variable is auto-set by Render when OIDC is enabled.
	AWSWebIdentityTokenFile string `required:"false" split_words:"true"`

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

	if config.AWSWebIdentityTokenFile != "" {
		logger.FromContext(ctx).Info("Using OIDC authentication")
		awscfg.Credentials = aws.NewCredentialsCache(stscreds.NewWebIdentityRoleProvider(
			sts.NewFromConfig(awscfg),
			config.AWSRoleARN,
			stscreds.IdentityTokenFile(config.AWSWebIdentityTokenFile),
		))
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
