variable "environment" {
  type = string
}

locals {
  set = {
    set1 = {
      name  = "service.type"
      value = "LoadBalancer"
    }
    set2 = {
      name  = "replicaCount"
      value = "2"
    }
    set3 = {
      name  = "ingress.enabled"
      value = "true"
    }
    set4 = {
      name  = "environment"
      value = var.environment
    }
  }
}
resource "helm_release" "nginx" {
  name       = "my-nginx"
  chart      = "nginx"
  repository = "https://charts.bitnami.com/bitnami"
  version    = "13.2.12"
  dynamic "set" {
    for_each = local.set
    content {
      name  = set.value.name
      value = set.value.value
    }
  }
}
