
data "aws_caller_identity" "current" {}

resource "aws_s3_bucket" "render_audit_logs" {
  bucket = var.aws_s3_bucket_name
}

resource "aws_s3_bucket_public_access_block" "render_audit_logs" {
  bucket = aws_s3_bucket.render_audit_logs.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "render_audit_logs" {
  bucket = aws_s3_bucket.render_audit_logs.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_policy" "render_audit_logs" {
  bucket = aws_s3_bucket.render_audit_logs.id
  policy = jsonencode({
    "Version" : "2012-10-17",
    "Statement" : [
      {
        Sid = "AllowAuditLogUpload",
        Effect = "Allow",
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:user/render-audit-log-processor"
        },
        Action = [
          "s3:ListBucket",
          "s3:PutObject",
          "s3:GetObject",
        ],
        Resource = [
          "arn:aws:s3:::${aws_s3_bucket.render_audit_logs.id}",
          "arn:aws:s3:::${aws_s3_bucket.render_audit_logs.id}/*",
        ]
      },
      {
          Sid    = "DenyUnencryptedObjectUploads",
          Effect = "Deny",
          Principal = "*",
          Action = "s3:PutObject",
          Resource = "arn:aws:s3:::${aws_s3_bucket.render_audit_logs.id}/*",
          Condition = var.aws_s3_use_kms ? {
            StringEquals = {
              "s3:x-amz-server-side-encryption" = "aws:kms"
            },
            Null = {
              "s3:x-amz-server-side-encryption-aws-kms-key-id" = "true"
            }
          } : {
            StringNotEquals = {
              "s3:x-amz-server-side-encryption" = "AES256"
            }
          }
        }

    ],
  })
}

resource "aws_s3_bucket_server_side_encryption_configuration" "render_audit_logs" {
    bucket = aws_s3_bucket.render_audit_logs.id
    rule {
      bucket_key_enabled = true
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }
