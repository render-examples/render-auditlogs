variable "aws_s3_bucket_name" {
    type = string
}

variable "aws_s3_bucket_key_enabled" {
  type = bool
  default = false
}

variable "aws_s3_kms_key_id" {
  type = string
  default = ""
}

variable "aws_s3_use_kms" {
  type = bool
  default = false
}

variable "aws_iam_role_name" {
  type = string
  default = "render-audit-log-processor"
  description = "Name of the IAM role the Cron Job assumes via OIDC"
}

variable "render_deployment_workspace_id" {
  type = string
  description = "Render workspace ID (tea-xxx) where the Cron Job is deployed; used to build the OIDC issuer URL (oidc.render.com/<workspace-id>)"
}

variable "aws_oidc_provider_arn" {
  type = string
  default = ""
  description = "ARN of an existing AWS IAM OIDC provider for Render. If empty, one is created."
}

variable "render_api_key" {
  type = string
  sensitive = true
}

variable "render_organization_id" {
    type = string
    default = ""
    description = "Render organization id (enterprise only)"
}

variable "render_workspace_ids" {
  type = list(string)
  default = []
  description = "Comma seperated string of workspace ids"
}

variable "render_cronjob_name" {
  type = string
  default = "render-auditlogs"
  description = "Name of the cron job"
}

variable "render_cronjob_schedule" {
  type = string
  default = "1/15 * * * *"
  description = "Schedule to run the sync"
}

variable "render_cronjob_plan" {
  type = string
  default = "starter"
  description = "Plan of the cronjob"
}

variable "render_cronjob_region" {
  type = string
  default = "oregon"
  description = "Region to deploy the cronjob"
}

variable "render_project_name" {
  type = string
  default = "audit-logs"
  description = "Name of project"
}
