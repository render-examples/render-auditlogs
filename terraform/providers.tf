terraform {
  required_providers {
    render = {
      source = "render-oss/render"
      version = "1.8.0"
    }

    aws = {
      source  = "hashicorp/aws"
      version = ">=5.39.0, < 6.0.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

provider "render" {}
