# render-auditlogs

Export [Render](https://render.com) audit logs to an AWS S3 bucket.

## Overview

This project provides:

- A Go application that fetches audit logs from the Render API and uploads them to S3
- Terraform modules to deploy the infrastructure on both AWS and Render
- Automatic scheduling via a Render Cron Job that runs every 15 minutes by default

Supports both workspace-level and organization-level (Enterprise) audit logs.

## Prerequisites

- Render workspace on Organization or Enterprise plan
- [Render API Key](https://dashboard.render.com/u/settings) (create from Account Settings). The Render API key must be a User account which is:
  - An Admin in every Workspace that will be tracked
  - An Owner of the Oranization (Enterprise Plan)
- Render Owner ID (`tea-xxx`) — workspace where the Cron Job will be deployed
- [Terraform](https://www.terraform.io/downloads) >= 1.0
- AWS account with permissions to create S3 buckets and IAM users

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/render-examples/render-auditlogs.git
cd render-auditlogs/terraform
```

### 2. Configure authentication

Set up authentication for both providers:

```bash
# AWS - use one of these methods:
export AWS_PROFILE=your-profile
# or
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

# Render - both are required for the Terraform provider
export RENDER_API_KEY=your-render-api-key
export RENDER_OWNER_ID=tea-xxxxx
```

### 3. Deploy with Terraform

```bash
terraform init
terraform apply \
  -var="aws_s3_bucket_name=your-audit-logs-bucket" \
  -var="render_api_key=${RENDER_API_KEY}" \
  -var='render_workspace_ids=["tea-xxxxx", "tea-yyyyy"]'
```

For Enterprise customers with organization-level audit logs:

```bash
terraform apply \
  -var="aws_s3_bucket_name=your-audit-logs-bucket" \
  -var="render_api_key=${RENDER_API_KEY}" \
  -var="render_organization_id=org-xxxxx" \
  -var='render_workspace_ids=["tea-xxxxx", "tea-yyyyy"]'
```

## Terraform Variables

| Variable                    | Required | Default                      | Description                                            |
| --------------------------- | -------- | ---------------------------- | ------------------------------------------------------ |
| `aws_s3_bucket_name`        | Yes      | -                            | Name of the S3 bucket to create for storing audit logs |
| `render_api_key`            | Yes      | -                            | Render API key for accessing audit logs                |
| `render_workspace_ids`      | No       | `[]`                         | List of workspace IDs to fetch audit logs from         |
| `render_organization_id`    | No       | `""`                         | Organization ID for Enterprise audit logs              |
| `aws_iam_user_name`         | No       | `render-audit-log-processor` | Name of the IAM user created for S3 access             |
| `aws_s3_bucket_key_enabled` | No       | `false`                      | Enable S3 bucket key to reduce KMS calls               |
| `aws_s3_kms_key_id`         | No       | `""`                         | ARN for KMS key to use for encryption                  |
| `aws_s3_use_kms`            | No       | `false`                      | Use KMS for encryption (instead of SSE-S3)             |
| `render_cronjob_name`       | No       | `render-auditlogs`           | Name of the Render Cron Job                            |
| `render_cronjob_schedule`   | No       | `1/15 * * * *`               | Cron schedule (default: every 15 minutes)              |
| `render_cronjob_plan`       | No       | `starter`                    | Render plan for the Cron Job                           |
| `render_cronjob_region`     | No       | `oregon`                     | Region to deploy the Cron Job                          |
| `render_project_name`       | No       | `audit-logs`                 | Name of the Render project                             |

*Note*: If you use a KMS key, confirm that the AWS IAM User is setup with the User Permissions for the key.

Example:
```
{
	"Version": "2012-10-17",
	"Id": "default",
	"Statement": [
		{
			"Sid": "Allow use of the key",
			"Effect": "Allow",
			"Principal": {
				"AWS": "arn:aws:iam::12345:user/render-audit-log-processor"
			},
			"Action": [
				"kms:Encrypt",
				"kms:Decrypt",
				"kms:ReEncrypt*",
				"kms:GenerateDataKey*",
				"kms:DescribeKey"
			],
			"Resource": "*"
		}
	]
}
```

## Architecture

The Terraform configuration creates:

**AWS Resources:**

- S3 bucket (versioned, encrypted, public access blocked)
- IAM user with S3 write permissions

**Render Resources:**

- Project
- Cron Job (builds from this repo)

## Local Development

To run the application locally:

1. Create a `.env` file:

```bash
WORKSPACE_IDS=tea-xxxxx,tea-yyyyy
ORGANIZATION_ID=org-xxxxx  # Optional, for Enterprise
S3_BUCKET=your-bucket-name
RENDER_API_KEY=your-api-key
AWS_ACCESS_KEY_ID=your-aws-key
AWS_SECRET_ACCESS_KEY=your-aws-secret
AWS_REGION=us-west-2

# Optional: KMS encryption settings (defaults to SSE-S3 if not set)
S3_USE_KMS=true
S3_KMS_KEY_ID=arn:aws:kms:us-west-2:123456789012:key/your-key-id  # Optional
S3_BUCKET_KEY_ENABLED=true  # Optional
```

2. Run the application:

```bash
go run main.go
```

## S3 Object Structure

Path format (Hive-style partitioning, gzip compressed):

```
s3://your-bucket/
  ├── workspace=tea-xxxxx/
  │   └── year=2024/
  │       └── month=1/
  │           └── day=15/
  │               └── audit-logs-2024-01-15_10-30-00.json.gz
  └── organization=org-xxxxx/
      └── year=2024/
          └── month=1/
              └── day=15/
                  └── audit-logs-2024-01-15_10-30-00.json.gz
```

## Integration with Panther SIEM

1. Create a custom log type in Panther with the schema below
2. Add an S3 log source pointing to your audit-logs bucket
3. Configure S3 event notifications to send object-create events to Panther

Panther schema:

```yaml
fields:
  - name: auditLog
    required: true
    type: object
    fields:
      - name: actor
        type: object
        fields:
          - name: email
            type: string
            indicators:
              - email
          - name: id
            type: string
          - name: type
            type: string
      - name: event
        type: string
      - name: id
        type: string
      - name: metadata
        type: json
      - name: status
        type: string
      - name: timestamp
        type: timestamp
        isEventTime: true
        timeFormats:
          - rfc3339
  - name: cursor
    required: true
    type: string
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -am 'Add my feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a Pull Request

Tests run automatically on PRs via GitHub Actions.

## Security

To report a security vulnerability, email security@render.com. Do not open a public issue.

## License

See [LICENSE](LICENSE) for details.
