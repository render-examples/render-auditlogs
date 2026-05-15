variable "aws_s3_bucket_name" {
    type = string
    default = ""
}

variable "aws_iam_role_name" {
  type = string
  default = "render-audit-log-processor"
}

variable "aws_s3_use_kms" {
  type    = bool
  default = false
}

variable "render_deployment_workspace_id" {
  type = string
}

variable "aws_oidc_provider_arn" {
  type = string
  default = ""
}

variable "render_cron_job_service_id" {
  type = string
  description = "Render Cron Job service ID (crn-xxx); pins the OIDC trust policy sub claim."
}
