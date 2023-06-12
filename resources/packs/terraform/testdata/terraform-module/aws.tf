provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias  = "replica"
  region = "eu-west-1"
}

module "remote-state-s3-backend" {
  source  = "nozaq/remote-state-s3-backend/aws"
  version = "1.4.0"

  providers = {
    aws         = aws
    aws.replica = aws.replica
  }

  enable_replication = false

  noncurrent_version_transitions = [
    {
      days          = 5
      storage_class = "GLACIER"
    }
  ]

  noncurrent_version_expiration = {
    days = 30
  }

  tags = {
    Module = "0-bootstrap-terraform"
  }
}