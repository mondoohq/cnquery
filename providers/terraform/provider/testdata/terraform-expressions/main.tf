// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Fixture covering HCL expression types that previously logged
// "unknown type" warnings or returned silent empty values:
// - ForExpr (tuple form):           [for x in y : x.id]
// - ForExpr (object form):          { for k in v : k => ... }
// - IndexExpr:                      m[k]
// - BinaryOpExpr:                   a == b, a % 2
// - ConditionalExpr (unbound vars): cond ? a : b
// - RelativeTraversalExpr:          (...).field
// - SplatExpr:                      list[*].field

locals {
  name = "${var.customer}-${var.stage}-${var.environment}-${var.instance_name}"

  subnet_id_by_az_suffix = {
    for zone in ["a", "b"] :
    zone => one([for subnet_id in data.aws_subnets.ec2.ids : subnet_id if endswith(data.aws_subnet.ec2[subnet_id].availability_zone, zone)])
  }

  az_suffix = var.availability_zone == "account_based" ? (data.aws_caller_identity.current.account_id % 2 == 0 ? "a" : "b") : var.availability_zone == "random" ? "a" : var.availability_zone
  subnet_id = var.availability_zone == "random" ? random_shuffle.ec2.result[0] : local.subnet_id_by_az_suffix[local.az_suffix]

  ami_id = var.disaster_recovery_mode ? var.disaster_recovery_ami_id : data.aws_ami.shared_image.id

  instance_ids = data.aws_instances.all[*].id
}

module "ec2_instance" {
  source  = "terraform-aws-modules/ec2-instance/aws"
  version = "6.3.0"

  name                   = local.name
  vpc_security_group_ids = [for sg in data.aws_security_group.ec2 : sg.id]
  ami                    = local.ami_id
  subnet_id              = local.subnet_id
}
