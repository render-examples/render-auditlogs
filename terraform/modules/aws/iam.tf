
resource "aws_iam_user" "log_processor" {
  name = var.aws_iam_user_name
}

resource "aws_iam_access_key" "log_processor" {
  user = aws_iam_user.log_processor.name
}

output "aws_access_key" {
  value = aws_iam_access_key.log_processor.id
  sensitive = true
}

output "aws_secret_access_key" {
  value = aws_iam_access_key.log_processor.secret
  sensitive = true
}
