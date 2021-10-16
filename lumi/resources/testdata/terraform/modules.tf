module "consul" {
  source  = "hashicorp/consul/aws"
  version = "0.0.5"

  servers = 3
}