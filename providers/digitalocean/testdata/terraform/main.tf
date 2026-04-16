terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

variable "do_token" {
  description = "DigitalOcean API token"
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key for droplet access"
  type        = string
  default     = ""
}

variable "region" {
  description = "DigitalOcean region"
  type        = string
  default     = "fra1"
}

provider "digitalocean" {
  token = var.do_token
}

# --- VPC ---

resource "digitalocean_vpc" "test" {
  name     = "mql-test-vpc"
  region   = var.region
  ip_range = "10.10.10.0/24"
}

# --- SSH Key (optional) ---

resource "digitalocean_ssh_key" "test" {
  count      = var.ssh_public_key != "" ? 1 : 0
  name       = "mql-test-key"
  public_key = var.ssh_public_key
}

# --- Project ---

resource "digitalocean_project" "test" {
  name        = "mql-test-project"
  description = "Test project for mql provider verification"
  purpose     = "Testing"
  environment = "Development"
}

# --- Droplet ---

resource "digitalocean_droplet" "test" {
  name       = "mql-test-droplet"
  size       = "s-1vcpu-512mb-10gb"
  image      = "ubuntu-24-04-x64"
  region     = var.region
  vpc_uuid   = digitalocean_vpc.test.id
  monitoring = true
  backups    = true
  tags       = ["mql-test", "automated"]
  ssh_keys   = var.ssh_public_key != "" ? [digitalocean_ssh_key.test[0].fingerprint] : []
}

# --- Firewall ---

resource "digitalocean_firewall" "test" {
  name        = "mql-test-firewall"
  droplet_ids = [digitalocean_droplet.test.id]
  tags        = ["mql-test"]

  # Allow SSH from a specific range (not 0.0.0.0/0 -- tests security policy)
  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["10.0.0.0/8"]
  }

  # Allow HTTP
  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # Allow HTTPS
  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # Allow all outbound
  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "icmp"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

# --- Volume ---

resource "digitalocean_volume" "test" {
  region      = var.region
  name        = "mql-test-volume"
  size        = 1
  description = "Test volume for mql provider verification"
  tags        = ["mql-test"]
}

resource "digitalocean_volume_attachment" "test" {
  droplet_id = digitalocean_droplet.test.id
  volume_id  = digitalocean_volume.test.id
}

# --- Database ---

resource "digitalocean_database_cluster" "test" {
  name       = "mql-test-db"
  engine     = "pg"
  version    = "17"
  size       = "db-s-1vcpu-1gb"
  region     = var.region
  node_count = 1
  tags       = ["mql-test"]

  maintenance_window {
    day  = "sunday"
    hour = "02:00:00"
  }
}

resource "digitalocean_database_firewall" "test" {
  cluster_id = digitalocean_database_cluster.test.id

  rule {
    type  = "droplet"
    value = digitalocean_droplet.test.id
  }
}

# --- Load Balancer ---

resource "digitalocean_loadbalancer" "test" {
  name                     = "mql-test-lb"
  region                   = var.region
  vpc_uuid                 = digitalocean_vpc.test.id
  redirect_http_to_https   = true
  enable_proxy_protocol    = false
  enable_backend_keepalive = true
  droplet_ids              = [digitalocean_droplet.test.id]

  forwarding_rule {
    entry_port      = 80
    entry_protocol  = "http"
    target_port     = 80
    target_protocol = "http"
  }

  forwarding_rule {
    entry_port      = 443
    entry_protocol  = "https"
    target_port     = 443
    target_protocol = "https"
    tls_passthrough = true
  }

  healthcheck {
    port     = 80
    protocol = "http"
    path     = "/"
  }
}

# --- Domain ---

resource "digitalocean_domain" "test" {
  name = "mql-test-${random_id.suffix.hex}.dev"
}

resource "random_id" "suffix" {
  byte_length = 4
}

resource "digitalocean_record" "a" {
  domain = digitalocean_domain.test.id
  type   = "A"
  name   = "www"
  value  = digitalocean_droplet.test.ipv4_address
  ttl    = 300
}

resource "digitalocean_record" "txt" {
  domain = digitalocean_domain.test.id
  type   = "TXT"
  name   = "@"
  value  = "mql-test-record"
  ttl    = 300
}

# --- Certificate (Let's Encrypt requires real domain, skip for testing) ---

# --- Outputs ---

output "droplet_ip" {
  value = digitalocean_droplet.test.ipv4_address
}

output "droplet_id" {
  value = digitalocean_droplet.test.id
}

output "database_host" {
  value     = digitalocean_database_cluster.test.host
  sensitive = true
}

output "lb_ip" {
  value = digitalocean_loadbalancer.test.ip
}

output "domain" {
  value = digitalocean_domain.test.name
}

output "vpc_id" {
  value = digitalocean_vpc.test.id
}
