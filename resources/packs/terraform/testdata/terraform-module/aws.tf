## dummy. we do not use bucket replication at all
provider "aws" {
  alias  = "replica"
  region = "eu-west-1"
}

module "remote-state-s3-backend" {
  providers = {
    aws         = aws
    aws.replica = aws.replica
  }

  source  = "nozaq/remote-state-s3-backend/aws"
  version = "1.4.0"

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