variable "aws_s3_bucket_name" {
  type = string
}

variable "aws_access_key" {
  type = string
  sensitive = true
}

variable "aws_secret_access_key" {
  type = string
  sensitive = true
}

variable "aws_region" {
  type = string
  default = "us-west-2"
}

variable "render_organization_id" {
    type = string
    default = ""
    description = "Render organization ID (enterprise only)"
}

variable "render_workspace_ids" {
  type = list(string)
  default = []
  description = "List of Render workspace IDs"
}

variable "render_api_key" {
  type = string
  sensitive = true
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
