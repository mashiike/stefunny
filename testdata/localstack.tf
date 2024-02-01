provider "aws" {
  region                      = "ap-northeast-1"
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
  s3_use_path_style           = true
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    apigateway     = "http://localhost:4566"
    cloudformation = "http://localhost:4566"
    cloudwatch     = "http://localhost:4566"
    cloudwatchlogs = "http://localhost:4566"
    dynamodb       = "http://localhost:4566"
    es             = "http://localhost:4566"
    firehose       = "http://localhost:4566"
    iam            = "http://localhost:4566"
    kinesis        = "http://localhost:4566"
    lambda         = "http://localhost:4566"
    route53        = "http://localhost:4566"
    redshift       = "http://localhost:4566"
    s3             = "http://localhost:4566"
    secretsmanager = "http://localhost:4566"
    ses            = "http://localhost:4566"
    sns            = "http://localhost:4566"
    sqs            = "http://localhost:4566"
    ssm            = "http://localhost:4566"
    stepfunctions  = "http://localhost:8083"
    sts            = "http://localhost:4566"
  }
}

terraform {
  required_version = "= 1.7.1"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "= 5.34.0"
    }
  }
  backend "local" {
    path = "terraform.tfstate"
  }
}

data "aws_caller_identity" "current" {
}

resource "aws_s3_bucket" "hoge" {
  bucket        = "test-hoge"
  force_destroy = false
}

resource "aws_cloudwatch_log_group" "test" {
  name = "test"
}

resource "aws_iam_role" "test" {
  name = "test"

  assume_role_policy = jsonencode(
    {
      "Version" = "2012-10-17",
      "Statement" = [
        {
          "Effect" = "Allow",
          "Principal" = {
            "Service" = "states.amazonaws.com"
          },
          "Action" = "sts:AssumeRole"
        }
      ]
  })
}
