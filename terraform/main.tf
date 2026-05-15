
data "aws_caller_identity" "current" {}

module "render" {
  source = "./modules/render-audit-logs"

  aws_account_id = data.aws_caller_identity.current.account_id
  aws_iam_role_name = var.aws_iam_role_name
  aws_s3_bucket_name = var.aws_s3_bucket_name
  aws_s3_bucket_key_enabled = var.aws_s3_bucket_key_enabled
  aws_s3_kms_key_id = var.aws_s3_kms_key_id
  aws_s3_use_kms = var.aws_s3_use_kms

  render_api_key = var.render_api_key
  render_organization_id = var.render_organization_id
  render_workspace_ids = var.render_workspace_ids
  render_project_name = var.render_project_name
  render_cronjob_name = var.render_cronjob_name
  render_cronjob_region = var.render_cronjob_region
  render_cronjob_plan = var.render_cronjob_plan
  render_cronjob_schedule = var.render_cronjob_schedule
}

module "aws" {
  source = "./modules/aws"

  aws_s3_bucket_name = var.aws_s3_bucket_name
  aws_iam_role_name = var.aws_iam_role_name
  aws_s3_use_kms = var.aws_s3_use_kms
  render_deployment_workspace_id = var.render_deployment_workspace_id
  aws_oidc_provider_arn = var.aws_oidc_provider_arn
  render_cron_job_service_id = module.render.cron_job_service_id
}
