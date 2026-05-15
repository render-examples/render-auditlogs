
locals {
  issuer_host          = "oidc.render.com"
  issuer_url           = "https://${local.issuer_host}/${var.render_deployment_workspace_id}"
  create_oidc_provider = var.aws_oidc_provider_arn == ""
  oidc_provider_arn    = var.aws_oidc_provider_arn != "" ? var.aws_oidc_provider_arn : aws_iam_openid_connect_provider.render[0].arn
}

resource "aws_iam_openid_connect_provider" "render" {
  count = local.create_oidc_provider ? 1 : 0

  url             = local.issuer_url
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = []
}

data "aws_iam_policy_document" "assume_role_with_oidc" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [local.oidc_provider_arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.issuer_host}/${var.render_deployment_workspace_id}:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "${local.issuer_host}/${var.render_deployment_workspace_id}:sub"
      values   = ["workspace:${var.render_deployment_workspace_id}:environment:*:service:${var.render_cron_job_service_id}"]
    }
  }
}

resource "aws_iam_role" "log_processor" {
  name               = var.aws_iam_role_name
  assume_role_policy = data.aws_iam_policy_document.assume_role_with_oidc.json
}

output "role_arn" {
  value = aws_iam_role.log_processor.arn
}
