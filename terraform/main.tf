
module "aws" {
  source = "./modules/aws"

  aws_s3_bucket_name = var.aws_s3_bucket_name
  aws_iam_user_name = var.aws_iam_user_name
  aws_s3_use_kms = var.aws_s3_use_kms
}

module "render" {
  source = "./modules/render-audit-logs"

  aws_access_key = module.aws.aws_access_key
  aws_secret_access_key = module.aws.aws_secret_access_key
  aws_s3_bucket_name = var.aws_s3_bucket_name

  render_api_key = var.render_api_key
  render_organization_id = var.render_organization_id
  render_workspace_ids = var.render_workspace_ids
  render_project_name = var.render_project_name
  render_cronjob_name = var.render_cronjob_name
  render_cronjob_region = var.render_cronjob_region
  render_cronjob_plan = var.render_cronjob_plan
  render_cronjob_schedule = var.render_cronjob_schedule
}
