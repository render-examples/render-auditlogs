
resource "render_cron_job" "render-audit-logs" {
  name               = var.render_cronjob_name
  plan               = var.render_cronjob_plan
  region             = var.render_cronjob_region
  schedule           = var.render_cronjob_schedule
  start_command      = "./render-auditlogs"

  runtime_source = {
    native_runtime = {
      auto_deploy   = true
      branch        = "main"
      build_command = "go build -tags netgo -ldflags '-s -w' -o render-auditlogs"
      repo_url = "https://github.com/render-examples/render-auditlogs"
      runtime  = "go"
    }
  }

  environment_id = render_project.audit-logs.environments["prod"].id

  env_vars = {
    "LOCAL" = { value = "false" },
    "AWS_ACCESS_KEY_ID" = {value = var.aws_access_key},
    "AWS_SECRET_ACCESS_KEY" = {value = var.aws_secret_access_key}
    "AWS_REGION" = {value = var.aws_region}
    "ORGANIZATION_ID" = {value = var.render_organization_id}
    "WORKSPACE_IDS" = { value = join(",", var.render_workspace_ids) }
    "RENDER_API_KEY" = { value = var.render_api_key }
    "S3_BUCKET" = { value = var.aws_s3_bucket_name }
    "S3_BUCKET_KEY_ENABLED" = { value = var.aws_s3_bucket_key_enabled }
    "S3_KMS_KEY_ID" = { value = var.aws_s3_kms_key_id }
    "S3_USE_KMS" = { value = var.aws_s3_use_kms }
  }
}

resource "render_project" "audit-logs" {
  name = var.render_project_name
  environments = {
    "prod" : {
      name : "prod",
      protected_status : "protected"
    },
  }
}
