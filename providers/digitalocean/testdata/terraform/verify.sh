#!/bin/bash
# Verify the mql DigitalOcean provider against live infrastructure.
# Prerequisites:
#   - DIGITALOCEAN_TOKEN set
#   - cnspec/mql installed with digitalocean provider
#   - Terraform resources applied (see main.tf)
#
# Usage:
#   cd providers/digitalocean/testdata/terraform
#   terraform apply -var="do_token=$DIGITALOCEAN_TOKEN"
#   ./verify.sh
#   terraform destroy -var="do_token=$DIGITALOCEAN_TOKEN"

set -eo pipefail

if [ -z "${DIGITALOCEAN_TOKEN:-}" ]; then
  echo "Error: DIGITALOCEAN_TOKEN is not set"
  exit 1
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass=0
fail=0

run_query() {
  local desc="$1"
  local query="$2"
  local result
  local exit_code=0
  result=$(mql run digitalocean --token "$DIGITALOCEAN_TOKEN" -c "$query" 2>&1) || exit_code=$?

  # Filter out normal config loading message, check for real errors
  local filtered
  filtered=$(echo "$result" | grep -v "loaded configuration from")

  if [ "$exit_code" -ne 0 ] || echo "$filtered" | grep -qi "error occurred\|panic\|unable to"; then
    echo -e "${RED}FAIL${NC}: $desc"
    echo "  Query: $query"
    echo "$filtered" | head -5 | sed 's/^/  /'
    fail=$((fail + 1))
  else
    echo -e "${GREEN}PASS${NC}: $desc"
    pass=$((pass + 1))
  fi
}

echo "=== mql DigitalOcean Provider Verification ==="
echo ""

# Account
run_query "account info" "digitalocean.account { email status emailVerified }"

# Droplets
run_query "list droplets" "digitalocean.droplets { id name status region }"
run_query "droplet details" "digitalocean.droplets { publicIpv4 privateIpv4 vpcUuid tags features }"
run_query "droplet monitoring" "digitalocean.droplets { name monitoringEnabled backupsEnabled }"
run_query "droplet image" "digitalocean.droplets { name image }"

# Firewalls
run_query "list firewalls" "digitalocean.firewalls { id name status }"
run_query "firewall rules" "digitalocean.firewalls { name inboundRules outboundRules }"
run_query "firewall droplets" "digitalocean.firewalls { name dropletIds tags }"

# Databases
run_query "list databases" "digitalocean.databases { id name engine version status }"
run_query "database details" "digitalocean.databases { region numNodes privateNetworkUuid tags }"
run_query "database firewall" "digitalocean.databases { name firewallRules }"
run_query "database maintenance" "digitalocean.databases { name maintenanceWindow }"
run_query "database users" "digitalocean.databases { name users { name role } }"
run_query "database replicas" "digitalocean.databases { name replicas { name status } }"
run_query "database pools" "digitalocean.databases { name pools { name mode } }"

# Domains
run_query "list domains" "digitalocean.domains { name ttl }"
run_query "domain records" "digitalocean.domains { name records { type name data } }"

# Volumes
run_query "list volumes" "digitalocean.volumes { id name sizeGigabytes region }"
run_query "volume details" "digitalocean.volumes { description filesystemType tags dropletIds }"

# Load Balancers
run_query "list LBs" "digitalocean.loadBalancers { id name status ip }"
run_query "LB config" "digitalocean.loadBalancers { redirectHttpToHttps enableProxyProtocol enableBackendKeepalive }"
run_query "LB rules" "digitalocean.loadBalancers { forwardingRules healthCheck }"

# VPCs
run_query "list VPCs" "digitalocean.vpcs { id name ipRange region default }"

# VPC Peerings
run_query "list VPC peerings" "digitalocean.vpcPeerings { id name status vpcIds }"

# Kubernetes
run_query "list K8s" "digitalocean.kubernetesClusters { id name version status }"
run_query "K8s node pools" "digitalocean.kubernetesClusters { name nodePools { name size count autoScale } }"

# Projects
run_query "list projects" "digitalocean.projects { id name environment isDefault }"

# SSH Keys
run_query "list SSH keys" "digitalocean.sshKeys { id name fingerprint }"

# Certificates
run_query "list certificates" "digitalocean.certificates { id name state type }"

# Container Registry
run_query "registry info" "digitalocean.registry { name region subscriptionTier }"
run_query "registry repos" "digitalocean.registryRepositories { name tagCount }"

# Reserved IPs
run_query "reserved IPs" "digitalocean.reservedIPs { ip region locked dropletId }"

# App Platform
run_query "list apps" "digitalocean.apps { id name liveUrl activeDeploymentStatus }"

# Monitoring
run_query "alert policies" "digitalocean.alertPolicies { uuid type enabled description }"

# Uptime Checks
run_query "uptime checks" "digitalocean.uptimeChecks { id name type target enabled }"

# CDN
run_query "CDN endpoints" "digitalocean.cdnEndpoints { id origin endpoint ttl }"

# Tags
run_query "tags" "digitalocean.tags { name resourceCount }"

# Spaces Keys
run_query "spaces keys" "digitalocean.spacesKeys { name accessKey grants }"

echo ""
echo "=== Results: ${pass} passed, ${fail} failed ==="

if [ "$fail" -gt 0 ]; then
  exit 1
fi
