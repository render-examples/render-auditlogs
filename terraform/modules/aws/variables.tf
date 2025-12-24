variable "aws_s3_bucket_name" {
    type = string
    default = ""
}

variable "aws_iam_user_name" {
  type = string
  default = "render-audit-log-processor"
}

variable "aws_s3_use_kms" {
  type    = bool
  default = false
}
