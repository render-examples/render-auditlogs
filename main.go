package main

import (
	"context"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/renderinc/render-auditlogs/pkg/auditlogs"
	"github.com/renderinc/render-auditlogs/pkg/aws"
	"github.com/renderinc/render-auditlogs/pkg/env"
	"github.com/renderinc/render-auditlogs/pkg/logger"
	"github.com/renderinc/render-auditlogs/pkg/processor"
	"github.com/renderinc/render-auditlogs/pkg/render"
)

const (
	renderAPIBaseURL = "https://api.render.com/v1"
)

func main() {
	ctx := context.Background()
	ctx, l := logger.New(ctx)

	var cfg env.Config
	err := env.LoadConfig(ctx, &cfg)
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Create S3 uploader
	uploader, err := aws.NewUploaderWithOptions(ctx, s3.NewFromConfig(cfg.AWSConfig), cfg.S3Bucket, cfg.AWSRegion, aws.UploaderOptions{
		UseKMS:           cfg.S3UseKMS,
		KMSKeyID:         cfg.S3KMSKeyID,
		BucketKeyEnabled: cfg.S3BucketKeyEnabled,
	})
	if err != nil {
		log.Fatal("Error creating S3 uploader:", err)
	}

	client := render.NewClient(renderAPIBaseURL, cfg.RenderAPIKey)

	workspaceLogs := auditlogs.NewWorkspaceSvc(client)
	organizationLogs := auditlogs.NewOrganizationSvc(client)

	semaphore := make(chan int, 5)
	var wg sync.WaitGroup

	for _, workspaceID := range cfg.WorkspaceIDS {
		semaphore <- 1
		wg.Add(1)

		go func(workspaceID string) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			ctx, l := logger.With(ctx, "workspaceID", workspaceID)

			l.Info("processing workspace")
			err := processor.NewLogProcessor(
				uploader, workspaceLogs,
			).Process(ctx, workspaceID)

			if err != nil {
				l.Error("Error processing workspace", "error", err)
			}

		}(workspaceID)
	}

	if cfg.OrganizationID != "" {
		semaphore <- 1
		wg.Add(1)
		go func(organizationID string) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			ctx, l := logger.With(ctx, "organizationID", cfg.OrganizationID)
			l.Info("processing enterprise")
			err = processor.NewLogProcessor(
				uploader, organizationLogs,
			).Process(ctx, cfg.OrganizationID)

			if err != nil {
				l.Error("Error processing organization audit logs", "error", err)
			}
		}(cfg.OrganizationID)
	}

	wg.Wait()
	l.Info("all workspaces processed")
}
