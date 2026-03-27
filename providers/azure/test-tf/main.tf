####################################################################
# Terraform config to exercise all new Azure security fields
#
# Usage:
#   terraform init && terraform apply
#
# Cost notes:
#   - Most resources are cheap/free tier.
#   - AKS + App Gateway are expensive (~$3-5/hr combined).
#     Comment them out if you only need to test other resources.
#   - Destroy promptly: terraform destroy
####################################################################

terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy    = true
      recover_soft_deleted_key_vaults = true
    }
  }
}

provider "azuread" {}

data "azurerm_client_config" "current" {}

locals {
  prefix   = "mqltest"
  location = "eastus"
  tags = {
    purpose = "mql-security-field-testing"
  }
}

resource "azurerm_resource_group" "test" {
  name     = "${local.prefix}-rg"
  location = local.location
  tags     = local.tags
}

# ─────────────────────────────────────────────────────
# Key Vault (premium SKU = HSM-backed, tests skuName)
# ─────────────────────────────────────────────────────
resource "azurerm_key_vault" "test" {
  name                       = "${local.prefix}-kv"
  location                   = azurerm_resource_group.test.location
  resource_group_name        = azurerm_resource_group.test.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "premium"
  purge_protection_enabled   = true
  soft_delete_retention_days = 7
  tags                       = local.tags

  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = data.azurerm_client_config.current.object_id

    key_permissions = [
      "Get", "List", "Create", "Delete", "Recover",
      "WrapKey", "UnwrapKey", "GetRotationPolicy",
    ]
    secret_permissions      = ["Get", "List", "Set", "Delete", "Recover"]
    certificate_permissions = ["Get", "List"]
  }
}

# CMK key for storage, SQL, etc.
resource "azurerm_key_vault_key" "cmk" {
  name         = "${local.prefix}-cmk"
  key_vault_id = azurerm_key_vault.test.id
  key_type     = "RSA"
  key_size     = 2048
  key_opts     = ["wrapKey", "unwrapKey"]
}

# ─────────────────────────────────────────────────────
# User-Assigned Managed Identity (for CMK access)
# ─────────────────────────────────────────────────────
resource "azurerm_user_assigned_identity" "cmk" {
  name                = "${local.prefix}-cmk-identity"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_key_vault_access_policy" "cmk_identity" {
  key_vault_id = azurerm_key_vault.test.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = azurerm_user_assigned_identity.cmk.principal_id

  key_permissions = ["Get", "WrapKey", "UnwrapKey"]
}

# ─────────────────────────────────────────────────────
# VNet + Subnet (for VNet rules, NIC, etc.)
# ─────────────────────────────────────────────────────
resource "azurerm_virtual_network" "test" {
  name                = "${local.prefix}-vnet"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  address_space       = ["10.0.0.0/16"]
  tags                = local.tags
}

resource "azurerm_subnet" "default" {
  name                 = "default"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test.name
  address_prefixes     = ["10.0.1.0/24"]
  service_endpoints    = ["Microsoft.AzureCosmosDB", "Microsoft.Sql", "Microsoft.Storage"]
}

resource "azurerm_subnet" "appgw" {
  name                 = "appgw"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test.name
  address_prefixes     = ["10.0.2.0/24"]
}

# ─────────────────────────────────────────────────────
# Storage Account (CMK encryption, tests encryptionKeySource + encryptionKey)
# ─────────────────────────────────────────────────────
resource "azurerm_storage_account" "platform" {
  name                     = "${local.prefix}platform"
  resource_group_name      = azurerm_resource_group.test.name
  location                 = azurerm_resource_group.test.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  tags                     = local.tags
}

resource "azurerm_storage_account" "cmk" {
  name                     = "${local.prefix}cmk"
  resource_group_name      = azurerm_resource_group.test.name
  location                 = azurerm_resource_group.test.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  tags                     = local.tags

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.cmk.id]
  }

  customer_managed_key {
    key_vault_key_id          = azurerm_key_vault_key.cmk.id
    user_assigned_identity_id = azurerm_user_assigned_identity.cmk.id
  }

  depends_on = [azurerm_key_vault_access_policy.cmk_identity]
}

# ─────────────────────────────────────────────────────
# Cosmos DB (CMK, backup policy, VNet rules)
# ─────────────────────────────────────────────────────
resource "azurerm_cosmosdb_account" "test" {
  name                = "${local.prefix}-cosmos"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  offer_type          = "Standard"
  kind                = "GlobalDocumentDB"
  tags                = local.tags

  consistency_policy {
    consistency_level = "Session"
  }

  geo_location {
    location          = azurerm_resource_group.test.location
    failover_priority = 0
  }

  is_virtual_network_filter_enabled = true

  virtual_network_rule {
    id                                   = azurerm_subnet.default.id
    ignore_missing_vnet_service_endpoint = false
  }

  backup {
    type                = "Periodic"
    interval_in_minutes = 240
    retention_in_hours  = 8
    storage_redundancy  = "Local"
  }
}

# ─────────────────────────────────────────────────────
# SQL Server + Database (encryption protector, AD-only auth)
# ─────────────────────────────────────────────────────
resource "azurerm_mssql_server" "test" {
  name                         = "${local.prefix}-sql"
  resource_group_name          = azurerm_resource_group.test.name
  location                     = azurerm_resource_group.test.location
  version                      = "12.0"
  administrator_login          = "sqladmin"
  administrator_login_password = "P@ssw0rd1234!"
  tags                         = local.tags

  azuread_administrator {
    login_username              = "AzureAD Admin"
    object_id                   = data.azurerm_client_config.current.object_id
    azuread_authentication_only = false
  }
}

resource "azurerm_mssql_database" "test" {
  name      = "${local.prefix}-db"
  server_id = azurerm_mssql_server.test.id
  sku_name  = "Basic"
}

# ─────────────────────────────────────────────────────
# PostgreSQL Flexible Server (backup config)
# ─────────────────────────────────────────────────────
resource "azurerm_postgresql_flexible_server" "test" {
  name                          = "${local.prefix}-pgflex"
  resource_group_name           = azurerm_resource_group.test.name
  location                      = azurerm_resource_group.test.location
  version                       = "16"
  administrator_login           = "pgadmin"
  administrator_password        = "P@ssw0rd1234!"
  storage_mb                    = 32768
  sku_name                      = "B_Standard_B1ms"
  zone                          = "1"
  public_network_access_enabled = true
  tags                          = local.tags

  backup_retention_days        = 14
  geo_redundant_backup_enabled = false
}

# ─────────────────────────────────────────────────────
# MySQL Flexible Server (backup config)
# ─────────────────────────────────────────────────────
resource "azurerm_mysql_flexible_server" "test" {
  name                   = "${local.prefix}-mysqlflex"
  resource_group_name    = azurerm_resource_group.test.name
  location               = azurerm_resource_group.test.location
  administrator_login    = "mysqladmin"
  administrator_password = "P@ssw0rd1234!"
  sku_name               = "B_Standard_B1s"
  version                = "8.0.21"
  zone                   = "1"
  tags                   = local.tags

  backup_retention_days        = 14
  geo_redundant_backup_enabled = false
}

# ─────────────────────────────────────────────────────
# Redis Cache (basic tier, tests encryptionKey null path)
# ─────────────────────────────────────────────────────
resource "azurerm_redis_cache" "test" {
  name                          = "${local.prefix}-redis"
  location                      = azurerm_resource_group.test.location
  resource_group_name           = azurerm_resource_group.test.name
  capacity                      = 0
  family                        = "C"
  sku_name                      = "Basic"
  minimum_tls_version           = "1.2"
  public_network_access_enabled = true
  tags                          = local.tags
}

# ─────────────────────────────────────────────────────
# Web App with IP restrictions + managed identity
# ─────────────────────────────────────────────────────
resource "azurerm_service_plan" "test" {
  name                = "${local.prefix}-plan"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  os_type             = "Linux"
  sku_name            = "B1"
  tags                = local.tags
}

resource "azurerm_linux_web_app" "test" {
  name                = "${local.prefix}-webapp"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  service_plan_id     = azurerm_service_plan.test.id
  https_only          = true
  tags                = local.tags

  identity {
    type = "SystemAssigned"
  }

  site_config {
    ip_restriction_default_action = "Deny"

    ip_restriction {
      name       = "AllowOfficeIP"
      action     = "Allow"
      ip_address = "203.0.113.0/24"
      priority   = 100
    }

    ip_restriction {
      name       = "AllowVPN"
      action     = "Allow"
      ip_address = "198.51.100.0/24"
      priority   = 200
    }

    scm_ip_restriction_default_action = "Deny"

    scm_ip_restriction {
      name       = "AllowCICD"
      action     = "Allow"
      ip_address = "192.0.2.0/24"
      priority   = 100
    }
  }
}

# ─────────────────────────────────────────────────────
# Public IP (with zones + DDoS)
# ─────────────────────────────────────────────────────
resource "azurerm_public_ip" "test" {
  name                = "${local.prefix}-pip"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  allocation_method   = "Static"
  sku                 = "Standard"
  zones               = ["1", "2", "3"]
  tags                = local.tags
}

# ─────────────────────────────────────────────────────
# Network Interface (with IP config)
# ─────────────────────────────────────────────────────
resource "azurerm_network_interface" "test" {
  name                = "${local.prefix}-nic"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  tags                = local.tags

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.default.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.test.id
  }
}

# ─────────────────────────────────────────────────────
# Load Balancer (Standard SKU, tests skuTier)
# ─────────────────────────────────────────────────────
resource "azurerm_public_ip" "lb" {
  name                = "${local.prefix}-lb-pip"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.tags
}

resource "azurerm_lb" "test" {
  name                = "${local.prefix}-lb"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  sku                 = "Standard"
  tags                = local.tags

  frontend_ip_configuration {
    name                 = "frontend"
    public_ip_address_id = azurerm_public_ip.lb.id
  }
}

# ─────────────────────────────────────────────────────
# Application Gateway (SSL policy, tests sslPolicyType etc.)
# NOTE: ~$1.50/hr — comment out if not needed
# ─────────────────────────────────────────────────────
resource "azurerm_public_ip" "appgw" {
  name                = "${local.prefix}-appgw-pip"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.tags
}

resource "azurerm_application_gateway" "test" {
  name                = "${local.prefix}-appgw"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  tags                = local.tags

  sku {
    name     = "Standard_v2"
    tier     = "Standard_v2"
    capacity = 1
  }

  ssl_policy {
    policy_type          = "CustomV2"
    min_protocol_version = "TLSv1_2"
    cipher_suites = [
      "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
      "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
    ]
  }

  gateway_ip_configuration {
    name      = "gateway-ip"
    subnet_id = azurerm_subnet.appgw.id
  }

  frontend_port {
    name = "http"
    port = 80
  }

  frontend_ip_configuration {
    name                 = "frontend"
    public_ip_address_id = azurerm_public_ip.appgw.id
  }

  backend_address_pool {
    name = "backend"
  }

  backend_http_settings {
    name                  = "http-settings"
    cookie_based_affinity = "Disabled"
    port                  = 80
    protocol              = "Http"
    request_timeout       = 30
  }

  http_listener {
    name                           = "listener"
    frontend_ip_configuration_name = "frontend"
    frontend_port_name             = "http"
    protocol                       = "Http"
  }

  request_routing_rule {
    name                       = "rule"
    rule_type                  = "Basic"
    http_listener_name         = "listener"
    backend_address_pool_name  = "backend"
    backend_http_settings_name = "http-settings"
    priority                   = 100
  }
}

# ─────────────────────────────────────────────────────
# AKS Cluster (Standard tier, tests skuTier)
# NOTE: ~$2-3/hr — comment out if not needed
# ─────────────────────────────────────────────────────
resource "azurerm_kubernetes_cluster" "test" {
  name                = "${local.prefix}-aks"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  dns_prefix          = "${local.prefix}-aks"
  sku_tier            = "Standard"
  tags                = local.tags

  default_node_pool {
    name       = "default"
    node_count = 1
    vm_size    = "Standard_B2s"
  }

  identity {
    type = "SystemAssigned"
  }
}

# ─────────────────────────────────────────────────────
# Outputs
# ─────────────────────────────────────────────────────
output "resource_group" {
  value = azurerm_resource_group.test.name
}

output "key_vault_name" {
  value = azurerm_key_vault.test.name
}

output "cmk_key_id" {
  value = azurerm_key_vault_key.cmk.id
}
